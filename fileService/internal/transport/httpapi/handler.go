package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fileService/internal/domain/fileentity"
	"fileService/internal/service/fileservice"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const multipartBodyOverhead = 1 << 20

type FileService interface {
	Upload(ctx context.Context, request fileservice.UploadRequest) (*fileentity.File, error)
	GetMetadata(ctx context.Context, fileID uuid.UUID) (*fileentity.File, error)
	GetContent(ctx context.Context, fileID uuid.UUID) (*fileservice.ContentResult, error)
	GetDownloadLink(ctx context.Context, fileID uuid.UUID) (*fileservice.DownloadLink, error)
	Delete(ctx context.Context, fileID uuid.UUID) error
}

type Handler struct {
	service       FileService
	maxFileSize   int64
	requestTimout time.Duration
}

func NewHandler(service FileService, maxFileSize int64, requestTimeout time.Duration) *Handler {
	return &Handler{
		service:       service,
		maxFileSize:   maxFileSize,
		requestTimout: requestTimeout,
	}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimout)
	defer cancel()

	r.Body = http.MaxBytesReader(w, r.Body, h.maxFileSize+multipartBodyOverhead)

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		writeError(w, mapMultipartError(err))
		return
	}
	defer file.Close()

	responseFile, err := h.service.Upload(ctx, fileservice.UploadRequest{
		Filename:     fileHeader.Filename,
		DeclaredMime: fileHeader.Header.Get("Content-Type"),
		Size:         fileHeader.Size,
		Content:      file,
		Uploader:     firstNonEmpty(r.FormValue("uploader"), r.Header.Get("X-Uploader")),
		OwnerService: firstNonEmpty(r.FormValue("owner_service"), r.Header.Get("X-Owner-Service")),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, uploadResponse{
		File: toFileDTO(responseFile),
		Links: linksDTO{
			Metadata: metadataURL(responseFile.ID),
			Content:  contentURL(responseFile.ID),
			Download: downloadURL(responseFile.ID),
		},
	})
}

func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimout)
	defer cancel()

	fileID, err := parseFileID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	file, err := h.service.GetMetadata(ctx, fileID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, metadataResponse{
		File: toFileDTO(file),
		Links: linksDTO{
			Metadata: metadataURL(file.ID),
			Content:  contentURL(file.ID),
			Download: downloadURL(file.ID),
		},
	})
}

func (h *Handler) GetContent(w http.ResponseWriter, r *http.Request) {
	fileID, err := parseFileID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.service.GetContent(r.Context(), fileID)
	if err != nil {
		writeError(w, err)
		return
	}
	defer result.Body.Close()

	w.Header().Set("Content-Type", result.File.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(result.File.Size, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", result.File.OriginalFilename))
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, result.Body); err != nil && !errors.Is(err, context.Canceled) {
		return
	}
}

func (h *Handler) GetDownloadLink(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimout)
	defer cancel()

	fileID, err := parseFileID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	link, err := h.service.GetDownloadLink(ctx, fileID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, downloadResponse{
		URL:              link.URL,
		ExpiresInSeconds: int64(link.ExpiresInSeconds),
	})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.requestTimout)
	defer cancel()

	fileID, err := parseFileID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := h.service.Delete(ctx, fileID); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseFileID(r *http.Request) (uuid.UUID, error) {
	rawID := mux.Vars(r)["id"]
	fileID, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		return uuid.Nil, fileentity.ErrInvalidFile
	}
	return fileID, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
