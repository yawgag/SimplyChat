package s3storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"fileService/internal/domain/fileentity"
	"fileService/internal/storage/objectstorage"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	client           *minio.Client
	bucket           string
	operationTimeout time.Duration
	maxRetries       int
}

func New(endpoint, accessKey, secretKey, bucket string, useSSL bool, operationTimeout time.Duration, maxRetries int) (*Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: operationTimeout,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("init s3 client: %w", err)
	}

	return &Storage{
		client:           client,
		bucket:           bucket,
		operationTimeout: operationTimeout,
		maxRetries:       maxRetries,
	}, nil
}

func (s *Storage) PutObject(ctx context.Context, params objectstorage.PutObjectParams) error {
	return s.withRetry(ctx, func(ctx context.Context) error {
		if _, err := params.Content.Seek(0, io.SeekStart); err != nil {
			return fileentity.ErrInvalidFile
		}

		_, err := s.client.PutObject(ctx, s.bucket, params.Key, params.Content, params.Size, minio.PutObjectOptions{
			ContentType: params.ContentType,
		})
		if err != nil {
			return mapError(err)
		}

		return nil
	})
}

func (s *Storage) GetObject(ctx context.Context, key string) (*objectstorage.Object, error) {
	var object *minio.Object
	var info minio.ObjectInfo

	err := s.withRetryNoTimeout(ctx, func(ctx context.Context) error {
		var err error
		object, err = s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
		if err != nil {
			return mapError(err)
		}

		info, err = object.Stat()
		if err != nil {
			object.Close()
			return mapError(err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &objectstorage.Object{
		Body:        object,
		Size:        info.Size,
		ContentType: info.ContentType,
	}, nil
}

func (s *Storage) DeleteObject(ctx context.Context, key string) error {
	return s.withRetry(ctx, func(ctx context.Context) error {
		err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
		if err != nil {
			return mapError(err)
		}
		return nil
	})
}

func (s *Storage) PresignGetObject(ctx context.Context, key string, expires time.Duration) (string, error) {
	var signedURL string

	err := s.withRetry(ctx, func(ctx context.Context) error {
		urlValue, err := s.client.PresignedGetObject(ctx, s.bucket, key, expires, url.Values{})
		if err != nil {
			return mapError(err)
		}
		signedURL = urlValue.String()
		return nil
	})
	if err != nil {
		return "", err
	}

	return signedURL, nil
}

func (s *Storage) withRetry(ctx context.Context, operation func(context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt < s.maxRetries; attempt++ {
		opCtx, cancel := context.WithTimeout(ctx, s.operationTimeout)
		err := operation(opCtx)
		cancel()
		if err == nil {
			return nil
		}

		lastErr = err
		if errors.Is(err, fileentity.ErrFileNotFound) {
			return err
		}
	}

	if lastErr == nil {
		lastErr = fileentity.ErrStorageUnavailable
	}

	return lastErr
}

func (s *Storage) withRetryNoTimeout(ctx context.Context, operation func(context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt < s.maxRetries; attempt++ {
		err := operation(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
		if errors.Is(err, fileentity.ErrFileNotFound) {
			return err
		}
	}

	if lastErr == nil {
		lastErr = fileentity.ErrStorageUnavailable
	}

	return lastErr
}

func mapError(err error) error {
	response := minio.ToErrorResponse(err)
	if response.Code == "NoSuchKey" || response.StatusCode == http.StatusNotFound {
		return fileentity.ErrFileNotFound
	}
	if response.Code != "" {
		return fileentity.ErrStorageUnavailable
	}
	if errors.Is(err, io.EOF) {
		return fileentity.ErrInvalidFile
	}
	return fileentity.ErrStorageUnavailable
}
