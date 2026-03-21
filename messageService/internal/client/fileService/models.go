package fileservice

import (
	"errors"
	"io"
	"time"
)

var (
	ErrValidation   = errors.New("file service validation error")
	ErrUnavailable  = errors.New("file service unavailable")
	ErrNotFound     = errors.New("file service not found")
	ErrBadResponse  = errors.New("file service bad response")
	ErrInvalidInput = errors.New("file service invalid input")
)

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type UploadRequest struct {
	Filename     string
	Content      ReadSeekCloser
	Uploader     string
	OwnerService string
}

type UploadedFile struct {
	ID               string    `json:"id"`
	OriginalFilename string    `json:"original_filename"`
	MimeType         string    `json:"mime_type"`
	Size             int64     `json:"size"`
	CreatedAt        time.Time `json:"created_at"`
	Uploader         string    `json:"uploader,omitempty"`
	OwnerService     string    `json:"owner_service,omitempty"`
}

type uploadResponse struct {
	File struct {
		ID               string    `json:"id"`
		OriginalFilename string    `json:"original_filename"`
		MimeType         string    `json:"mime_type"`
		Size             int64     `json:"size"`
		CreatedAt        time.Time `json:"created_at"`
		Uploader         string    `json:"uploader,omitempty"`
		OwnerService     string    `json:"owner_service,omitempty"`
	} `json:"file"`
}

type apiErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
