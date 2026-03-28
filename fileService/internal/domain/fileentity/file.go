package fileentity

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID               uuid.UUID  `json:"id"`
	ObjectKey        string     `json:"object_key"`
	OriginalFilename string     `json:"original_filename"`
	MimeType         string     `json:"mime_type"`
	Size             int64      `json:"size"`
	CreatedAt        time.Time  `json:"created_at"`
	DeletedAt        *time.Time `json:"-"`
	Uploader         string     `json:"uploader,omitempty"`
	OwnerService     string     `json:"owner_service,omitempty"`
}
