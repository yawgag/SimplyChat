package httpapi

import (
	"time"

	"fileService/internal/domain/fileentity"

	"github.com/google/uuid"
)

type fileDTO struct {
	ID               uuid.UUID `json:"id"`
	ObjectKey        string    `json:"object_key"`
	OriginalFilename string    `json:"original_filename"`
	MimeType         string    `json:"mime_type"`
	Size             int64     `json:"size"`
	CreatedAt        time.Time `json:"created_at"`
	Uploader         string    `json:"uploader,omitempty"`
	OwnerService     string    `json:"owner_service,omitempty"`
}

type linksDTO struct {
	Metadata string `json:"metadata"`
	Content  string `json:"content"`
	Download string `json:"download,omitempty"`
}

type uploadResponse struct {
	File  fileDTO  `json:"file"`
	Links linksDTO `json:"links"`
}

type metadataResponse struct {
	File  fileDTO  `json:"file"`
	Links linksDTO `json:"links"`
}

type downloadResponse struct {
	URL              string `json:"url"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func toFileDTO(file *fileentity.File) fileDTO {
	return fileDTO{
		ID:               file.ID,
		ObjectKey:        file.ObjectKey,
		OriginalFilename: file.OriginalFilename,
		MimeType:         file.MimeType,
		Size:             file.Size,
		CreatedAt:        file.CreatedAt,
		Uploader:         file.Uploader,
		OwnerService:     file.OwnerService,
	}
}

func metadataURL(fileID uuid.UUID) string {
	return "/files/" + fileID.String()
}

func contentURL(fileID uuid.UUID) string {
	return "/files/" + fileID.String() + "/content"
}

func downloadURL(fileID uuid.UUID) string {
	return "/files/" + fileID.String() + "/download"
}
