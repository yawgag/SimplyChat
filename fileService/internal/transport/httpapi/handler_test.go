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
	"github.com/gorilla/mux"
)

type fileServiceStub struct {
	uploadFunc     func(ctx context.Context, request fileservice.UploadRequest) (*fileentity.File, error)
	getContentFunc func(ctx context.Context, fileID uuid.UUID) (*fileservice.ContentResult, error)
}

func (s *fileServiceStub) Upload(ctx context.Context, request fileservice.UploadRequest) (*fileentity.File, error) {
	return s.uploadFunc(ctx, request)
}

func (s *fileServiceStub) GetMetadata(ctx context.Context, fileID uuid.UUID) (*fileentity.File, error) {
	return nil, nil
}

func (s *fileServiceStub) GetContent(ctx context.Context, fileID uuid.UUID) (*fileservice.ContentResult, error) {
	if s.getContentFunc != nil {
		return s.getContentFunc(ctx, fileID)
	}
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

func TestDownloadStreamsFileAsAttachment(t *testing.T) {
	fileID := uuid.New()
	handler := NewHandler(&fileServiceStub{
		getContentFunc: func(ctx context.Context, requestedID uuid.UUID) (*fileservice.ContentResult, error) {
			if requestedID != fileID {
				t.Fatalf("unexpected file id: %s", requestedID)
			}
			return &fileservice.ContentResult{
				File: &fileentity.File{
					ID:               fileID,
					OriginalFilename: "photo.png",
					MimeType:         "image/png",
					Size:             4,
				},
				Body: io.NopCloser(strings.NewReader("test")),
			}, nil
		},
	}, 1024, time.Second)

	request := httptest.NewRequest(http.MethodGet, "/files/"+fileID.String()+"/download", nil)
	request = muxSetVars(request, map[string]string{"id": fileID.String()})

	recorder := httptest.NewRecorder()
	handler.GetDownloadLink(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Disposition"); got != `attachment; filename="photo.png"` {
		t.Fatalf("unexpected content disposition: %q", got)
	}
	if got := recorder.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if body := recorder.Body.String(); body != "test" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func muxSetVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}
