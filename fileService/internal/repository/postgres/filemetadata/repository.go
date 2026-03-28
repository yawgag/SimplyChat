package filemetadata

import (
	"context"
	"errors"
	"fmt"

	"fileService/internal/domain/fileentity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, file *fileentity.File) error {
	query := `insert into file_metadata(id, object_key, original_filename, mime_type, size, uploader, owner_service)
		values ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query, file.ID, file.ObjectKey, file.OriginalFilename, file.MimeType, file.Size, nullableString(file.Uploader), nullableString(file.OwnerService))
	if err != nil {
		return fmt.Errorf("create file metadata: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*fileentity.File, error) {
	query := `select id, object_key, original_filename, mime_type, size, created_at, deleted_at, coalesce(uploader, ''), coalesce(owner_service, '')
		from file_metadata
		where id = $1 and deleted_at is null`

	file := &fileentity.File{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&file.ID,
		&file.ObjectKey,
		&file.OriginalFilename,
		&file.MimeType,
		&file.Size,
		&file.CreatedAt,
		&file.DeletedAt,
		&file.Uploader,
		&file.OwnerService,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fileentity.ErrFileNotFound
		}
		return nil, fmt.Errorf("get file metadata: %w", err)
	}

	return file, nil
}

func (r *Repository) GetByIDForDeletion(ctx context.Context, id uuid.UUID) (*fileentity.File, error) {
	query := `select id, object_key, original_filename, mime_type, size, created_at, deleted_at, coalesce(uploader, ''), coalesce(owner_service, '')
		from file_metadata
		where id = $1`

	file := &fileentity.File{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&file.ID,
		&file.ObjectKey,
		&file.OriginalFilename,
		&file.MimeType,
		&file.Size,
		&file.CreatedAt,
		&file.DeletedAt,
		&file.Uploader,
		&file.OwnerService,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fileentity.ErrFileNotFound
		}
		return nil, fmt.Errorf("get file metadata for deletion: %w", err)
	}

	return file, nil
}

func (r *Repository) MarkDeleted(ctx context.Context, id uuid.UUID) error {
	query := `update file_metadata
		set deleted_at = now()
		where id = $1 and deleted_at is null`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark file metadata deleted: %w", err)
	}
	if result.RowsAffected() == 0 {
		return nil
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `delete from file_metadata where id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete file metadata: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fileentity.ErrFileNotFound
	}

	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
