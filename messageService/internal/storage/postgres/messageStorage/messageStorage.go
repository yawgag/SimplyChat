package messageStorage

//
//
//
//
// TODO
// this code has very bad logic, need to rewrite it later
//
// добавить нормализацию пользователей в чате.
// добавить проверку, что переписка происходит именно между этими пользователями.
//
//
//

import (
	"context"
	"errors"
	"fmt"
	"messageService/internal/models"
	"messageService/internal/storage/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrorChatDoesntExist error = errors.New("chat doens't exist")
)

type messageHandler struct {
	pool postgres.DBPool
}

type Handler interface {
	SaveMessage(ctx context.Context, msg *models.EventSendMessagePayload) error
	GetMessageHistory(ctx context.Context, startNum int, chatId int) ([]models.EventSendMessagePayload, error)
}

func NewMessageStorage(pool postgres.DBPool) Handler {
	out := &messageHandler{
		pool: pool,
	}
	return out
}

func (m *messageHandler) getChat(ctx context.Context, chatId int) (*models.Chat, error) {
	query := `select id, userA, userB
				from chat
				where id = $1`

	var chat models.Chat
	err := m.pool.QueryRow(ctx, query, chatId).Scan(&chat.Id, &chat.UserAId, &chat.UserBId)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, ErrorChatDoesntExist
	}
	if err != nil {
		return nil, fmt.Errorf("[getChat] error: %s", err)
	}
	return &chat, nil
}

func (m *messageHandler) createChat(ctx context.Context, userAId, userBid uuid.UUID) (int, error) {
	query := `insert into chat(userA, userb)
				values($1, $2)
				returning id`
	var chatId int

	err := m.pool.QueryRow(ctx, query, userAId, userBid).Scan(&chatId)
	if err != nil {
		return -1, fmt.Errorf("[createChat] error: %s", err)
	}

	return chatId, nil
}

// TODO my eyes...
func (m *messageHandler) SaveMessage(ctx context.Context, msg *models.EventSendMessagePayload) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("[SaveMessage] error: %s", err)
	}
	defer tx.Rollback(ctx)

	var chatId int

	_, err = m.getChat(ctx, msg.ChatId)
	if err != nil {
		if err == ErrorChatDoesntExist {
			createdId, err := m.createChat(ctx, msg.SenderId, msg.RecipientId)
			if err != nil {
				return fmt.Errorf("[SaveMessage] error: %s", err)
			}
			chatId = createdId
		} else {
			return fmt.Errorf("[SaveMessage] error: %s", err)
		}
	} else {
		chatId = msg.ChatId
	}

	query := `insert into message(chatId, senderId, recipientId, data)
				values ($1, $2, $3, $4)`

	_, err = m.pool.Exec(ctx, query, chatId, msg.SenderId, msg.RecipientId, msg.Content)
	if err != nil {
		return fmt.Errorf("[SaveMessage] error: %s", err)
	}
	return nil
}

func (m *messageHandler) GetMessageHistory(ctx context.Context, startNum int, chatId int) ([]models.EventSendMessagePayload, error) {
	query := `select id, senderId, recipientId, data, addedAt
				from message m
				where chatId = $1
				order by addedAt desc
				offset $2
				limit 100`

	rows, err := m.pool.Query(ctx, query, chatId, startNum)
	if err != nil {
		return nil, fmt.Errorf("[GetMessageHistory] error: %s", err)
	}
	defer rows.Close()

	var out []models.EventSendMessagePayload
	for rows.Next() {
		var row models.EventSendMessagePayload
		err := rows.Scan(
			&row.Id,
			&row.SenderId,
			&row.RecipientId,
			&row.Content,
			&row.CreatedAt,
		)
		if err != nil {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}
