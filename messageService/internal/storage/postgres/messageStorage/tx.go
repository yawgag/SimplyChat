package messageStorage

import (
	"context"
	"fmt"
	"messageService/internal/storage/postgres"
	"time"
)

func withTx(ctx context.Context, pool postgres.DBPool, fn func(tx postgres.Tx) error) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("[withTx] begin error: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("[withTx] commit error: %w", err)
	}

	return nil
}

func createMessageTx(ctx context.Context, tx postgres.Tx, chatID int, senderLogin string, kind string, content string) (int, time.Time, error) {
	query := `insert into message(chat_id, sender_login, kind, message_text)
				values ($1, $2, $3, $4)
				returning id, added_at`

	var (
		messageID int
		createdAt time.Time
	)

	err := tx.QueryRow(ctx, query, chatID, senderLogin, kind, nullableString(content)).Scan(&messageID, &createdAt)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("[createMessageTx] error: %w", err)
	}

	return messageID, createdAt, nil
}

func createAttachmentsTx(ctx context.Context, tx postgres.Tx, messageID int, attachments []FileAttachmentRecord) error {
	query := `insert into message_attachment(message_id, file_id, original_filename, mime_type, size, kind)
				values ($1, $2, $3, $4, $5, $6)`

	for _, attachment := range attachments {
		_, err := tx.Exec(
			ctx,
			query,
			messageID,
			attachment.FileID,
			attachment.OriginalFilename,
			attachment.MimeType,
			attachment.Size,
			attachment.Kind,
		)
		if err != nil {
			return fmt.Errorf("[createAttachmentsTx] error: %w", err)
		}
	}

	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
