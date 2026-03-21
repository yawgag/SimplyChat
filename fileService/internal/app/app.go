package app

import (
	"fmt"
	"net/http"
	"time"

	"fileService/internal/config"
	"fileService/internal/platform/postgres"
	"fileService/internal/repository/postgres/filemetadata"
	"fileService/internal/service/fileservice"
	"fileService/internal/storage/s3storage"
	"fileService/internal/transport/httpapi"
)

type App struct {
	server *http.Server
}

func NewApp() (*App, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	dbPool, err := postgres.InitDB(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}

	objectStorage, err := s3storage.New(
		cfg.S3Endpoint,
		cfg.S3AccessKey,
		cfg.S3SecretKey,
		cfg.S3Bucket,
		cfg.S3UseSSL,
		cfg.S3OperationTimeout,
		cfg.S3MaxRetries,
	)
	if err != nil {
		return nil, fmt.Errorf("init object storage: %w", err)
	}

	repository := filemetadata.NewRepository(dbPool)
	service := fileservice.New(repository, objectStorage, cfg.MaxFileSizeBytes, cfg.AllowedMimeTypes, cfg.PresignedURLTTL)
	handler := httpapi.NewHandler(service, cfg.MaxFileSizeBytes, cfg.RequestTimeout)
	router := httpapi.NewRouter(handler)

	server := &http.Server{
		Addr:         cfg.ServiceAddr,
		Handler:      router,
		ReadTimeout:  cfg.RequestTimeout,
		WriteTimeout: 0,
		IdleTimeout:  time.Minute,
	}

	return &App{server: server}, nil
}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}
