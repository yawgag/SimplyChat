package objectstorage

import (
	"context"
	"io"
	"time"
)

type PutObjectParams struct {
	Key         string
	Content     io.ReadSeeker
	Size        int64
	ContentType string
}

type Object struct {
	Body        io.ReadCloser
	Size        int64
	ContentType string
}

type Storage interface {
	PutObject(ctx context.Context, params PutObjectParams) error
	GetObject(ctx context.Context, key string) (*Object, error)
	DeleteObject(ctx context.Context, key string) error
	PresignGetObject(ctx context.Context, key string, expires time.Duration) (string, error)
}
