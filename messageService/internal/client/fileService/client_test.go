package fileservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type noopReadSeekCloser struct {
	*strings.Reader
}

func (n noopReadSeekCloser) Close() error { return nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestUploadSuccess(t *testing.T) {
	var seenUploader string
	var seenOwner string
	var seenFilename string
	var seenBody string

	client := &Client{
		baseURL: "http://example.com",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost {
					t.Fatalf("unexpected method: %s", r.Method)
				}
				if r.URL.Path != "/files" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}

				if err := r.ParseMultipartForm(32 << 20); err != nil {
					t.Fatalf("parse multipart: %v", err)
				}

				seenUploader = r.FormValue("uploader")
				seenOwner = r.FormValue("owner_service")

				file, header, err := r.FormFile("file")
				if err != nil {
					t.Fatalf("form file: %v", err)
				}
				defer file.Close()

				seenFilename = header.Filename
				raw, err := io.ReadAll(file)
				if err != nil {
					t.Fatalf("read file: %v", err)
				}
				seenBody = string(raw)

				body := bytes.NewBuffer(nil)
				_ = json.NewEncoder(body).Encode(map[string]any{
					"file": map[string]any{
						"id":                "file-1",
						"original_filename": "photo.png",
						"mime_type":         "image/png",
						"size":              4,
						"created_at":        time.Now().UTC(),
						"uploader":          "alice",
						"owner_service":     "messageService",
					},
				})

				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(body),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	uploaded, err := client.Upload(context.Background(), UploadRequest{
		Filename:     "photo.png",
		Content:      noopReadSeekCloser{strings.NewReader("test")},
		Uploader:     "alice",
		OwnerService: "messageService",
	})
	if err != nil {
		t.Fatalf("upload error: %v", err)
	}

	if uploaded.ID != "file-1" || uploaded.OriginalFilename != "photo.png" || uploaded.MimeType != "image/png" {
		t.Fatalf("unexpected uploaded file: %+v", uploaded)
	}
	if seenUploader != "alice" || seenOwner != "messageService" || seenFilename != "photo.png" || seenBody != "test" {
		t.Fatalf("unexpected request values: uploader=%q owner=%q filename=%q body=%q", seenUploader, seenOwner, seenFilename, seenBody)
	}
}

func TestUploadValidationMapping(t *testing.T) {
	client := &Client{
		baseURL: "http://example.com",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := bytes.NewBufferString(`{"error":{"code":"mime_type_not_allowed","message":"file mime type is not allowed"}}`)
				return &http.Response{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(body),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	_, err := client.Upload(context.Background(), UploadRequest{
		Filename:     "photo.png",
		Content:      noopReadSeekCloser{strings.NewReader("test")},
		Uploader:     "alice",
		OwnerService: "messageService",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestDeleteNotFoundMapping(t *testing.T) {
	client := &Client{
		baseURL: "http://example.com",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewBuffer(nil)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	err := client.Delete(context.Background(), "file-1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}
