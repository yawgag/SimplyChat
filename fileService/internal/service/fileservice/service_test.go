package fileservice

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"fileService/internal/domain/fileentity"
	"fileService/internal/storage/objectstorage"

	"github.com/google/uuid"
)

type repositoryStub struct {
	file               *fileentity.File
	createErr          error
	getErr             error
	getForDeleteErr    error
	markDeletedErr     error
	deleteErr          error
	createCalled       bool
	getCalled          bool
	getForDeleteCalled bool
	markDeletedCalled  bool
	deleteCalled       bool
}

func (r *repositoryStub) Create(ctx context.Context, file *fileentity.File) error {
	r.createCalled = true
	r.file = file
	return r.createErr
}

func (r *repositoryStub) GetByID(ctx context.Context, id uuid.UUID) (*fileentity.File, error) {
	r.getCalled = true
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.file, nil
}

func (r *repositoryStub) GetByIDForDeletion(ctx context.Context, id uuid.UUID) (*fileentity.File, error) {
	r.getForDeleteCalled = true
	if r.getForDeleteErr != nil {
		return nil, r.getForDeleteErr
	}
	return r.file, nil
}

func (r *repositoryStub) MarkDeleted(ctx context.Context, id uuid.UUID) error {
	r.markDeletedCalled = true
	return r.markDeletedErr
}

func (r *repositoryStub) Delete(ctx context.Context, id uuid.UUID) error {
	r.deleteCalled = true
	return r.deleteErr
}

type storageStub struct {
	putErr         error
	getErr         error
	deleteErr      error
	presignErr     error
	putCalled      bool
	deleteCalled   bool
	lastPutPayload []byte
}

func (s *storageStub) PutObject(ctx context.Context, params objectstorage.PutObjectParams) error {
	s.putCalled = true
	payload, err := io.ReadAll(params.Content)
	if err != nil {
		return err
	}
	s.lastPutPayload = payload
	return s.putErr
}

func (s *storageStub) GetObject(ctx context.Context, key string) (*objectstorage.Object, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return &objectstorage.Object{
		Body: io.NopCloser(bytes.NewReader(nil)),
	}, nil
}

func (s *storageStub) DeleteObject(ctx context.Context, key string) error {
	s.deleteCalled = true
	return s.deleteErr
}

func (s *storageStub) PresignGetObject(ctx context.Context, key string, expires time.Duration) (string, error) {
	if s.presignErr != nil {
		return "", s.presignErr
	}
	return "https://example.com/file", nil
}

func TestUploadRejectsMimeTypeNotAllowed(t *testing.T) {
	repository := &repositoryStub{}
	storage := &storageStub{}
	service := New(repository, storage, 1024, []string{"image/png"}, time.Minute)

	_, err := service.Upload(context.Background(), UploadRequest{
		Filename:     "photo.jpg",
		DeclaredMime: "image/jpeg",
		Size:         int64(len("hello")),
		Content:      bytes.NewReader([]byte("hello")),
	})

	if !errors.Is(err, fileentity.ErrMimeTypeNotAllowed) {
		t.Fatalf("expected ErrMimeTypeNotAllowed, got %v", err)
	}
	if repository.createCalled {
		t.Fatal("repository should not be called when mime validation fails")
	}
	if storage.putCalled {
		t.Fatal("storage should not be called when mime validation fails")
	}
}

func TestUploadReturnsJoinedErrorWhenMetadataCreateAndCleanupFail(t *testing.T) {
	repository := &repositoryStub{createErr: errors.New("db down")}
	storage := &storageStub{deleteErr: fileentity.ErrStorageUnavailable}
	service := New(repository, storage, 1024, []string{"text/plain"}, time.Minute)

	_, err := service.Upload(context.Background(), UploadRequest{
		Filename:     "note.txt",
		DeclaredMime: "text/plain",
		Size:         int64(len("hello")),
		Content:      bytes.NewReader([]byte("hello")),
	})

	if !errors.Is(err, fileentity.ErrMetadataUnavailable) {
		t.Fatalf("expected metadata error, got %v", err)
	}
	if !errors.Is(err, fileentity.ErrStorageUnavailable) {
		t.Fatalf("expected joined cleanup storage error, got %v", err)
	}
	if !storage.deleteCalled {
		t.Fatal("expected cleanup delete to be attempted")
	}
}

func TestDeleteMarksMetadataBeforeStorageDelete(t *testing.T) {
	fileID := uuid.New()
	repository := &repositoryStub{
		file: &fileentity.File{
			ID:        fileID,
			ObjectKey: "files/test-key",
		},
	}
	storage := &storageStub{deleteErr: fileentity.ErrStorageUnavailable}
	service := New(repository, storage, 1024, []string{"text/plain"}, time.Minute)

	err := service.Delete(context.Background(), fileID)

	if !errors.Is(err, fileentity.ErrStorageUnavailable) {
		t.Fatalf("expected storage error, got %v", err)
	}
	if !repository.getForDeleteCalled {
		t.Fatal("expected GetByIDForDeletion to be called")
	}
	if !repository.markDeletedCalled {
		t.Fatal("expected MarkDeleted to be called before storage delete")
	}
	if repository.deleteCalled {
		t.Fatal("hard delete should not be called when storage delete fails")
	}
}

func TestDeleteTreatsMissingObjectAsRecoverable(t *testing.T) {
	fileID := uuid.New()
	repository := &repositoryStub{
		file: &fileentity.File{
			ID:        fileID,
			ObjectKey: "files/test-key",
		},
	}
	storage := &storageStub{deleteErr: fileentity.ErrFileNotFound}
	service := New(repository, storage, 1024, []string{"text/plain"}, time.Minute)

	err := service.Delete(context.Background(), fileID)
	if err != nil {
		t.Fatalf("expected delete to succeed when object is already missing, got %v", err)
	}
	if !repository.deleteCalled {
		t.Fatal("expected hard delete after missing object")
	}
}
