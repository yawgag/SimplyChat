package fileservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"fileService/internal/domain/fileentity"
	"fileService/internal/storage/objectstorage"

	"github.com/google/uuid"
)

const sniffBufferSize = 512

type MetadataRepository interface {
	Create(ctx context.Context, file *fileentity.File) error
	GetByID(ctx context.Context, id uuid.UUID) (*fileentity.File, error)
	GetByIDForDeletion(ctx context.Context, id uuid.UUID) (*fileentity.File, error)
	MarkDeleted(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type Service struct {
	repository      MetadataRepository
	storage         objectstorage.Storage
	maxFileSize     int64
	allowedMimeType map[string]struct{}
	presignTTL      time.Duration
}

type UploadRequest struct {
	Filename     string
	DeclaredMime string
	Size         int64
	Content      io.Reader
	Uploader     string
	OwnerService string
}

type DownloadLink struct {
	URL              string `json:"url"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
}

type ContentResult struct {
	File *fileentity.File
	Body io.ReadCloser
}

func New(repository MetadataRepository, storage objectstorage.Storage, maxFileSize int64, allowedMimeTypes []string, presignTTL time.Duration) *Service {
	allowed := make(map[string]struct{}, len(allowedMimeTypes))
	for _, mimeType := range allowedMimeTypes {
		allowed[strings.ToLower(strings.TrimSpace(mimeType))] = struct{}{}
	}

	return &Service{
		repository:      repository,
		storage:         storage,
		maxFileSize:     maxFileSize,
		allowedMimeType: allowed,
		presignTTL:      presignTTL,
	}
}

func (s *Service) Upload(ctx context.Context, request UploadRequest) (*fileentity.File, error) {
	if request.Content == nil {
		return nil, fileentity.ErrInvalidFile
	}
	if request.Size <= 0 {
		return nil, fileentity.ErrInvalidFile
	}
	if request.Size > s.maxFileSize {
		return nil, fileentity.ErrFileTooLarge
	}

	filename := sanitizeFilename(request.Filename)
	if filename == "" {
		return nil, fileentity.ErrInvalidFile
	}

	tempFile, head, err := spoolToTempFile(request.Content, request.Size)
	if err != nil {
		return nil, fileentity.ErrInvalidFile
	}
	defer func() {
		tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		return nil, fileentity.ErrInvalidFile
	}
	if len(head) == 0 {
		return nil, fileentity.ErrInvalidFile
	}

	mimeType := detectMimeType(head, request.DeclaredMime)
	if _, ok := s.allowedMimeType[mimeType]; !ok {
		return nil, fileentity.ErrMimeTypeNotAllowed
	}

	fileID := uuid.New()
	file := &fileentity.File{
		ID:               fileID,
		ObjectKey:        buildObjectKey(fileID, filename),
		OriginalFilename: filename,
		MimeType:         mimeType,
		Size:             request.Size,
		Uploader:         strings.TrimSpace(request.Uploader),
		OwnerService:     strings.TrimSpace(request.OwnerService),
	}

	if err := s.storage.PutObject(ctx, objectstorage.PutObjectParams{
		Key:         file.ObjectKey,
		Content:     tempFile,
		Size:        file.Size,
		ContentType: file.MimeType,
	}); err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, file); err != nil {
		cleanupErr := s.storage.DeleteObject(ctx, file.ObjectKey)
		if cleanupErr != nil {
			return nil, errors.Join(fileentity.ErrMetadataUnavailable, cleanupErr)
		}
		return nil, fileentity.ErrMetadataUnavailable
	}

	storedFile, err := s.repository.GetByID(ctx, file.ID)
	if err != nil {
		return nil, fileentity.ErrMetadataUnavailable
	}

	return storedFile, nil
}

func (s *Service) GetMetadata(ctx context.Context, fileID uuid.UUID) (*fileentity.File, error) {
	file, err := s.repository.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (s *Service) GetContent(ctx context.Context, fileID uuid.UUID) (*ContentResult, error) {
	file, err := s.repository.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	object, err := s.storage.GetObject(ctx, file.ObjectKey)
	if err != nil {
		return nil, err
	}

	return &ContentResult{
		File: file,
		Body: object.Body,
	}, nil
}

func (s *Service) GetDownloadLink(ctx context.Context, fileID uuid.UUID) (*DownloadLink, error) {
	file, err := s.repository.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	url, err := s.storage.PresignGetObject(ctx, file.ObjectKey, s.presignTTL)
	if err != nil {
		return nil, err
	}

	return &DownloadLink{
		URL:              url,
		ExpiresInSeconds: int64(s.presignTTL / time.Second),
	}, nil
}

func (s *Service) Delete(ctx context.Context, fileID uuid.UUID) error {
	file, err := s.repository.GetByIDForDeletion(ctx, fileID)
	if err != nil {
		return err
	}

	if err := s.repository.MarkDeleted(ctx, fileID); err != nil {
		return fileentity.ErrMetadataUnavailable
	}
	if err := s.storage.DeleteObject(ctx, file.ObjectKey); err != nil && !errors.Is(err, fileentity.ErrFileNotFound) {
		return err
	}
	if err := s.repository.Delete(ctx, fileID); err != nil {
		return fileentity.ErrMetadataUnavailable
	}

	return nil
}

func detectMimeType(head []byte, declared string) string {
	detected := strings.ToLower(http.DetectContentType(head))
	if detected != "" && detected != "application/octet-stream" {
		return normalizeMimeType(detected)
	}
	return normalizeMimeType(declared)
}

func normalizeMimeType(value string) string {
	mediaType, _, err := mime.ParseMediaType(strings.ToLower(strings.TrimSpace(value)))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(value))
	}
	return mediaType
}

func sanitizeFilename(filename string) string {
	cleaned := strings.TrimSpace(filepath.Base(filename))
	cleaned = strings.ReplaceAll(cleaned, "\x00", "")
	if cleaned == "." || cleaned == "" {
		return ""
	}
	return cleaned
}

func buildObjectKey(fileID uuid.UUID, filename string) string {
	extension := strings.ToLower(path.Ext(filename))
	if len(extension) > 16 {
		extension = ""
	}

	now := time.Now().UTC()
	return fmt.Sprintf("files/%04d/%02d/%02d/%s%s", now.Year(), now.Month(), now.Day(), fileID.String(), extension)
}

func spoolToTempFile(source io.Reader, expectedSize int64) (*os.File, []byte, error) {
	tempFile, err := os.CreateTemp("", "file-service-upload-*")
	if err != nil {
		return nil, nil, err
	}

	cleanupOnError := func(cause error) (*os.File, []byte, error) {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return nil, nil, cause
	}

	headBuffer := make([]byte, sniffBufferSize)
	readBytes, err := io.ReadFull(source, headBuffer)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return cleanupOnError(err)
	}

	head := headBuffer[:readBytes]
	if len(head) == 0 {
		return cleanupOnError(fileentity.ErrInvalidFile)
	}

	if _, err := tempFile.Write(head); err != nil {
		return cleanupOnError(err)
	}

	writtenTail, err := io.Copy(tempFile, source)
	if err != nil {
		return cleanupOnError(err)
	}

	totalSize := int64(len(head)) + writtenTail
	if totalSize != expectedSize {
		return cleanupOnError(fileentity.ErrInvalidFile)
	}

	return tempFile, head, nil
}
