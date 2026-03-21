package fileservice

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	defaultRequestTimeout = 10 * time.Second
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Upload(ctx context.Context, req UploadRequest) (*UploadedFile, error) {
	if req.Content == nil || strings.TrimSpace(req.Filename) == "" {
		return nil, ErrInvalidInput
	}

	if _, err := req.Content.Seek(0, io.SeekStart); err != nil {
		return nil, ErrInvalidInput
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/files", pr)
	if err != nil {
		_ = pw.CloseWithError(err)
		return nil, ErrBadResponse
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	log.Printf("[messageService][fileClient][Upload] start filename=%q uploader=%q owner_service=%q target=%s", req.Filename, req.Uploader, req.OwnerService, c.baseURL+"/files")

	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("file", path.Base(req.Filename))
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		if _, err := io.Copy(part, req.Content); err != nil {
			_ = pw.CloseWithError(err)
			return
		}

		if err := writer.WriteField("uploader", req.Uploader); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if err := writer.WriteField("owner_service", req.OwnerService); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()

	response, err := c.httpClient.Do(request)
	if err != nil {
		log.Printf("[messageService][fileClient][Upload] transport_error filename=%q err=%v", req.Filename, err)
		return nil, ErrUnavailable
	}
	defer response.Body.Close()
	log.Printf("[messageService][fileClient][Upload] response filename=%q status=%d", req.Filename, response.StatusCode)

	switch {
	case response.StatusCode == http.StatusCreated:
		var payload uploadResponse
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			return nil, ErrBadResponse
		}
		return &UploadedFile{
			ID:               payload.File.ID,
			OriginalFilename: payload.File.OriginalFilename,
			MimeType:         payload.File.MimeType,
			Size:             payload.File.Size,
			CreatedAt:        payload.File.CreatedAt,
			Uploader:         payload.File.Uploader,
			OwnerService:     payload.File.OwnerService,
		}, nil
	case response.StatusCode == http.StatusBadRequest ||
		response.StatusCode == http.StatusRequestEntityTooLarge ||
		response.StatusCode == http.StatusUnsupportedMediaType ||
		response.StatusCode == http.StatusUnprocessableEntity:
		log.Printf("[messageService][fileClient][Upload] validation_error filename=%q status=%d", req.Filename, response.StatusCode)
		return nil, ErrValidation
	case response.StatusCode >= 500:
		log.Printf("[messageService][fileClient][Upload] unavailable filename=%q status=%d", req.Filename, response.StatusCode)
		return nil, ErrUnavailable
	default:
		log.Printf("[messageService][fileClient][Upload] bad_response filename=%q status=%d", req.Filename, response.StatusCode)
		return nil, ErrBadResponse
	}
}

func (c *Client) Delete(ctx context.Context, fileID string) error {
	if strings.TrimSpace(fileID) == "" {
		return ErrInvalidInput
	}

	target := c.baseURL + "/files/" + url.PathEscape(fileID)
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, target, nil)
	if err != nil {
		return ErrBadResponse
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		log.Printf("[messageService][fileClient][Delete] transport_error file_id=%s err=%v", fileID, err)
		return ErrUnavailable
	}
	defer response.Body.Close()
	log.Printf("[messageService][fileClient][Delete] response file_id=%s status=%d", fileID, response.StatusCode)

	switch {
	case response.StatusCode == http.StatusNoContent:
		return nil
	case response.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case response.StatusCode >= 500:
		return ErrUnavailable
	default:
		return ErrBadResponse
	}
}
