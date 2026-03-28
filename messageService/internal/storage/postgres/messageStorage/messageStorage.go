package messageStorage

import (
	"context"
	"errors"
	"fmt"
	"messageService/internal/models"
	"messageService/internal/storage/postgres"
	"slices"
	"strings"
	"time"
)

const (
	messageKindText = "text"
	messageKindFile = "file"
)

var (
	ErrorChatDoesntExist error = errors.New("chat doens't exist")
)

type messageHandler struct {
	pool postgres.DBPool
}

type FileAttachmentRecord struct {
	FileID           string
	OriginalFilename string
	MimeType         string
	Size             int64
	Kind             string
}

type historyRow struct {
	messageID        int
	chatID           int
	senderLogin      string
	kind             string
	content          string
	createdAt        time.Time
	fileID           *string
	originalFilename *string
	mimeType         *string
	size             *int64
	attachmentKind   *string
}

type HistoryPageParams struct {
	ChatID int
	Limit  int
	Before *models.MessageHistoryCursor
}

type Handler interface {
	SaveMessage(ctx context.Context, msg *models.EventSendMessagePayload) (*models.EventSendMessagePayload, error)
	GetMessageHistory(ctx context.Context, params HistoryPageParams) (*models.EventSendMessageHistoryPayload, error)
	GetAllUserChats(ctx context.Context, login string) ([]models.EventChatPayload, error)
	CreateChat(ctx context.Context, maxMembers int) (int, error)
	AddUserToChat(ctx context.Context, chatId int, userLogin string) error
	GetAllChatMembers(ctx context.Context, chatId int) ([]string, error)
	ChatExists(ctx context.Context, chatId int) (bool, error)
	IsChatMember(ctx context.Context, chatId int, login string) (bool, error)
	UserHasAccessToFile(ctx context.Context, login string, fileID string) (bool, error)
	CreateFileMessage(ctx context.Context, chatId int, senderLogin string, attachments []FileAttachmentRecord) (*models.EventSendMessagePayload, error)
}

func NewMessageStorage(pool postgres.DBPool) Handler {
	return &messageHandler{
		pool: pool,
	}
}

func (m *messageHandler) CreateChat(ctx context.Context, maxMembers int) (int, error) {
	query := `insert into chat(max_members)
				values($1)
				returning id`
	var chatId int
	err := m.pool.QueryRow(ctx, query, maxMembers).Scan(&chatId)
	if err != nil {
		return -1, fmt.Errorf("[CreateChat] error: %s", err)
	}

	return chatId, nil
}

func (m *messageHandler) GetAllChatMembers(ctx context.Context, chatId int) ([]string, error) {
	query := `select user_login
				from chat_user
				where chat_id = $1`

	rows, err := m.pool.Query(ctx, query, chatId)
	if err != nil {
		return nil, fmt.Errorf("[GetAllChatMembers] error: %s", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var row string
		err := rows.Scan(
			&row,
		)
		if err != nil {
			continue
		}
		out = append(out, row)
	}

	return out, nil
}

func (m *messageHandler) AddUserToChat(ctx context.Context, chatId int, userLogin string) error {
	query := `insert into chat_user(chat_id, user_login)
				select $1, $2
				where (
					select count(*) from chat_user where chat_id = $1
				) < (
					select max_members from chat where id = $1
				)`

	res, err := m.pool.Exec(ctx, query, chatId, userLogin)
	if err != nil {
		return fmt.Errorf("[AddUserToChat] error: %s", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("[AddUserToChat] error: can't add user")
	}

	return nil
}

// tx method
func (m *messageHandler) chatIsExist(tx postgres.Tx, ctx context.Context, chatId int) error {
	query := `select exists(
					select 1
					from chat
					where id = $1)`

	var res bool
	err := tx.QueryRow(ctx, query, chatId).Scan(&res)
	if err != nil {
		return fmt.Errorf("[chatIsExist] error: %s", err)
	}
	if !res {
		return ErrorChatDoesntExist
	}
	return nil
}

func (m *messageHandler) GetAllUserChats(ctx context.Context, login string) ([]models.EventChatPayload, error) {
	query := `select chat_id
				from chat_user
				where user_login = $1`
	rows, err := m.pool.Query(ctx, query, login)
	if err != nil {
		return nil, fmt.Errorf("[GetAllUserChat] error: %s", err)
	}
	defer rows.Close()

	var out []models.EventChatPayload
	for rows.Next() {
		var row models.EventChatPayload
		err := rows.Scan(&row.ChatId)
		if err != nil {
			continue
		}
		out = append(out, row)
	}

	return out, nil
}

func (m *messageHandler) SaveMessage(ctx context.Context, msg *models.EventSendMessagePayload) (*models.EventSendMessagePayload, error) {
	query := `insert into message(chat_id, sender_login, kind, message_text)
				values ($1, $2, $3, $4)
				returning id, added_at`

	savedMessage := &models.EventSendMessagePayload{
		ChatId:      msg.ChatId,
		SenderLogin: msg.SenderLogin,
		Kind:        messageKindText,
		Content:     msg.Content,
		Attachments: []models.AttachmentPayload{},
	}

	err := m.pool.QueryRow(ctx, query, msg.ChatId, msg.SenderLogin, messageKindText, msg.Content).Scan(&savedMessage.Id, &savedMessage.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("[SaveMessage] error: %s", err)
	}
	return savedMessage, nil
}

func (m *messageHandler) GetMessageHistory(ctx context.Context, params HistoryPageParams) (*models.EventSendMessageHistoryPayload, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}

	query := `with paged_messages as (
				select id, chat_id, sender_login, kind, coalesce(message_text, '') as message_text, added_at
				from message
				where chat_id = $1
				  and (
					$2::timestamptz is null
					or (added_at, id) < ($2::timestamptz, coalesce($3::int, 0))
				  )
				order by added_at desc, id desc
				limit $4
			)
			select pm.id, pm.chat_id, pm.sender_login, pm.kind, pm.message_text, pm.added_at,
				   a.file_id, a.original_filename, a.mime_type, a.size, a.kind
			from paged_messages pm
			left join message_attachment a on a.message_id = pm.id
			order by pm.added_at desc, pm.id desc, a.id`

	var beforeCreatedAt *time.Time
	var beforeID *int
	if params.Before != nil {
		beforeCreatedAt = &params.Before.CreatedAt
		beforeID = &params.Before.ID
	}

	rows, err := m.pool.Query(ctx, query, params.ChatID, beforeCreatedAt, beforeID, limit+1)
	if err != nil {
		return nil, fmt.Errorf("[GetMessageHistory] error: %s", err)
	}
	defer rows.Close()

	var historyRows []historyRow
	for rows.Next() {
		var row historyRow
		err := rows.Scan(
			&row.messageID,
			&row.chatID,
			&row.senderLogin,
			&row.kind,
			&row.content,
			&row.createdAt,
			&row.fileID,
			&row.originalFilename,
			&row.mimeType,
			&row.size,
			&row.attachmentKind,
		)
		if err != nil {
			continue
		}
		historyRows = append(historyRows, row)
	}

	history := aggregateHistoryRows(historyRows)
	hasMore := len(history) > limit
	if hasMore {
		history = history[:limit]
	}
	reverseHistory(history)

	page := &models.EventSendMessageHistoryPayload{
		ChatId:  params.ChatID,
		Items:   history,
		HasMore: hasMore,
	}
	if hasMore && len(history) > 0 {
		oldest := history[0]
		page.NextCursor = &models.MessageHistoryCursor{
			CreatedAt: oldest.CreatedAt,
			ID:        oldest.Id,
		}
	}

	return page, nil
}

func aggregateHistoryRows(rows []historyRow) []models.EventSendMessagePayload {
	history := make([]models.EventSendMessagePayload, 0)
	byMessageID := make(map[int]int)

	for _, row := range rows {
		index, exists := byMessageID[row.messageID]
		if !exists {
			message := models.EventSendMessagePayload{
				Id:          row.messageID,
				ChatId:      row.chatID,
				SenderLogin: row.senderLogin,
				Kind:        row.kind,
				Content:     row.content,
				Attachments: []models.AttachmentPayload{},
				CreatedAt:   row.createdAt,
			}
			history = append(history, message)
			index = len(history) - 1
			byMessageID[row.messageID] = index
		}

		if row.fileID == nil {
			continue
		}

		attachment := models.AttachmentPayload{
			FileID:           *row.fileID,
			OriginalFilename: stringOrEmpty(row.originalFilename),
			MimeType:         stringOrEmpty(row.mimeType),
			Size:             int64OrZero(row.size),
			Kind:             stringOrEmpty(row.attachmentKind),
		}

		if !containsAttachment(history[index].Attachments, attachment.FileID) {
			history[index].Attachments = append(history[index].Attachments, attachment)
		}
	}

	return history
}

func containsAttachment(attachments []models.AttachmentPayload, fileID string) bool {
	return slices.ContainsFunc(attachments, func(attachment models.AttachmentPayload) bool {
		return attachment.FileID == fileID
	})
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func int64OrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func reverseHistory(history []models.EventSendMessagePayload) {
	for left, right := 0, len(history)-1; left < right; left, right = left+1, right-1 {
		history[left], history[right] = history[right], history[left]
	}
}

func (m *messageHandler) ChatExists(ctx context.Context, chatId int) (bool, error) {
	query := `select exists(select 1 from chat where id = $1)`

	var exists bool
	err := m.pool.QueryRow(ctx, query, chatId).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("[ChatExists] error: %s", err)
	}

	return exists, nil
}

func (m *messageHandler) IsChatMember(ctx context.Context, chatId int, login string) (bool, error) {
	query := `select exists(
				select 1
				from chat_user
				where chat_id = $1 and user_login = $2
			)`

	var exists bool
	err := m.pool.QueryRow(ctx, query, chatId, login).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("[IsChatMember] error: %s", err)
	}

	return exists, nil
}

func (m *messageHandler) UserHasAccessToFile(ctx context.Context, login string, fileID string) (bool, error) {
	query := `select exists(
				select 1
				from message_attachment ma
				join message m on m.id = ma.message_id
				join chat_user cu on cu.chat_id = m.chat_id
				where ma.file_id = $1::uuid and cu.user_login = $2
			)`

	var exists bool
	err := m.pool.QueryRow(ctx, query, strings.TrimSpace(fileID), login).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("[UserHasAccessToFile] error: %s", err)
	}

	return exists, nil
}

func (m *messageHandler) CreateFileMessage(ctx context.Context, chatId int, senderLogin string, attachments []FileAttachmentRecord) (*models.EventSendMessagePayload, error) {
	var payload *models.EventSendMessagePayload

	err := withTx(ctx, m.pool, func(tx postgres.Tx) error {
		messageID, createdAt, err := createMessageTx(ctx, tx, chatId, senderLogin, messageKindFile, "")
		if err != nil {
			return err
		}

		if err := createAttachmentsTx(ctx, tx, messageID, attachments); err != nil {
			return err
		}

		payload = &models.EventSendMessagePayload{
			Id:          messageID,
			ChatId:      chatId,
			SenderLogin: senderLogin,
			Kind:        messageKindFile,
			Content:     "",
			Attachments: make([]models.AttachmentPayload, 0, len(attachments)),
			CreatedAt:   createdAt,
		}
		for _, attachment := range attachments {
			payload.Attachments = append(payload.Attachments, models.AttachmentPayload{
				FileID:           attachment.FileID,
				OriginalFilename: attachment.OriginalFilename,
				MimeType:         attachment.MimeType,
				Size:             attachment.Size,
				Kind:             strings.ToLower(attachment.Kind),
			})
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("[CreateFileMessage] error: %w", err)
	}

	return payload, nil
}
