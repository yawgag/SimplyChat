package fileentity

import "errors"

var (
	ErrInvalidFile         = errors.New("invalid file")
	ErrFileTooLarge        = errors.New("file is too large")
	ErrMimeTypeNotAllowed  = errors.New("mime type is not allowed")
	ErrFileNotFound        = errors.New("file not found")
	ErrStorageUnavailable  = errors.New("storage unavailable")
	ErrMetadataUnavailable = errors.New("metadata unavailable")
)
