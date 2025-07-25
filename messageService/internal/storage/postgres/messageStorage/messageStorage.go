package messageStorage

import (
	"context"
	"errors"
	"fmt"
	"messageService/internal/models"
	"messageService/internal/storage/postgres"
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
	GetAllUserChats(ctx context.Context, login string) ([]models.EventChatPayload, error)
	CreateChat(ctx context.Context, maxMembers int) (int, error)
	AddUserToChat(ctx context.Context, chatId int, userLogin string) error
	GetAllChatMembers(ctx context.Context, chatId int) ([]string, error)
}

func NewMessageStorage(pool postgres.DBPool) Handler {
	out := &messageHandler{
		pool: pool,
	}
	return out
}

func (m *messageHandler) CreateChat(ctx context.Context, maxMembers int) (int, error) {
	query := `insert into chat(maxMembers)
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
	query := `select userLogin
				from chat_user
				where chatId = $1`

	rows, err := m.pool.Query(ctx, query, chatId)
	if err != nil {
		return nil, fmt.Errorf("[GetAllChatMembers] error: %s", err)
	}

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
	query := `insert into chat_user(chatId, userLogin)
				select $1, $2
				where (
					select count(*) from chat_user where chatId = $1
				) < (
					select maxMembers from chat where id = $1
				)`

	res, err := m.pool.Exec(ctx, query, chatId, userLogin)
	if res.RowsAffected() == 0 {
		return fmt.Errorf("[AddUserToChat] error: can't add user")
	}

	return err
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
	query := `select chatId
				from chat_user
				where userLogin = $1`
	rows, err := m.pool.Query(ctx, query, login)
	if err != nil {
		return nil, fmt.Errorf("[GetAllUserChat] error: %s", err)
	}

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

func (m *messageHandler) SaveMessage(ctx context.Context, msg *models.EventSendMessagePayload) error {
	query := `insert into message(chatId, senderLogin, data)
				values ($1, $2, $3)`

	_, err := m.pool.Exec(ctx, query, msg.ChatId, msg.SenderLogin, msg.Content)
	if err != nil {
		return fmt.Errorf("[SaveMessage] error: %s", err)
	}
	return nil
}

func (m *messageHandler) GetMessageHistory(ctx context.Context, startNum int, chatId int) ([]models.EventSendMessagePayload, error) {
	query := `select id, senderLogin, data, addedAt
				from message m
				where chatId = $1
				order by addedAt
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
			&row.SenderLogin,
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
