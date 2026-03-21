package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fileService/internal/domain/fileentity"
	"fileService/internal/service/fileservice"

	"github.com/google/uuid"
)

type fileServiceStub struct {
	uploadFunc func(ctx context.Context, request fileservice.UploadRequest) (*fileentity.File, error)
}

func (s *fileServiceStub) Upload(ctx context.Context, request fileservice.UploadRequest) (*fileentity.File, error) {
	return s.uploadFunc(ctx, request)
}

func (s *fileServiceStub) GetMetadata(ctx context.Context, fileID uuid.UUID) (*fileentity.File, error) {
	return nil, nil
}

func (s *fileServiceStub) GetContent(ctx context.Context, fileID uuid.UUID) (*fileservice.ContentResult, error) {
	return nil, nil
}

func (s *fileServiceStub) GetDownloadLink(ctx context.Context, fileID uuid.UUID) (*fileservice.DownloadLink, error) {
	return nil, nil
}

func (s *fileServiceStub) Delete(ctx context.Context, fileID uuid.UUID) error {
	return nil
}

func TestUploadResponseUsesStableDownloadRoute(t *testing.T) {
	fileID := uuid.New()
	handler := NewHandler(&fileServiceStub{
		uploadFunc: func(ctx context.Context, request fileservice.UploadRequest) (*fileentity.File, error) {
			return &fileentity.File{
				ID:               fileID,
				ObjectKey:        "files/2026/03/21/" + fileID.String() + ".png",
				OriginalFilename: "photo.png",
				MimeType:         "image/png",
				Size:             4,
				CreatedAt:        time.Unix(1, 0).UTC(),
			}, nil
		},
	}, 1024, time.Second)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "photo.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(part, strings.NewReader("test")); err != nil {
		t.Fatalf("write multipart body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/files", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	recorder := httptest.NewRecorder()
	handler.Upload(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", recorder.Code)
	}

	var response uploadResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	expectedDownload := "/files/" + fileID.String() + "/download"
	if response.Links.Download != expectedDownload {
		t.Fatalf("expected stable download route %q, got %q", expectedDownload, response.Links.Download)
	}
}
