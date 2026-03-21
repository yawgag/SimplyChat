package httpapi

import (
	"errors"
	"mime/multipart"
	"net/http"

	"fileService/internal/domain/fileentity"
)

func writeError(w http.ResponseWriter, err error) {
	status, code, message := mapError(err)
	writeJSON(w, status, errorResponse{
		Error: errorBody{
			Code:    code,
			Message: message,
		},
	})
}

func mapError(err error) (int, string, string) {
	switch {
	case errors.Is(err, fileentity.ErrInvalidFile):
		return http.StatusBadRequest, "invalid_file", "request contains invalid file data"
	case errors.Is(err, fileentity.ErrFileTooLarge):
		return http.StatusRequestEntityTooLarge, "file_too_large", "file size exceeds configured limit"
	case errors.Is(err, fileentity.ErrMimeTypeNotAllowed):
		return http.StatusUnsupportedMediaType, "mime_type_not_allowed", "file mime type is not allowed"
	case errors.Is(err, fileentity.ErrFileNotFound):
		return http.StatusNotFound, "file_not_found", "file was not found"
	case errors.Is(err, fileentity.ErrMetadataUnavailable):
		return http.StatusServiceUnavailable, "metadata_unavailable", "file metadata storage is unavailable"
	case errors.Is(err, fileentity.ErrStorageUnavailable):
		return http.StatusServiceUnavailable, "storage_unavailable", "file object storage is unavailable"
	default:
		return http.StatusInternalServerError, "internal_error", "internal server error"
	}
}

func mapMultipartError(err error) error {
	if errors.Is(err, http.ErrMissingFile) {
		return fileentity.ErrInvalidFile
	}
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return fileentity.ErrFileTooLarge
	}
	if errors.Is(err, multipart.ErrMessageTooLarge) {
		return fileentity.ErrFileTooLarge
	}
	return fileentity.ErrInvalidFile
}
