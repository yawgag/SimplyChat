package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	Uid           uuid.UUID
	Conn          *websocket.Conn
	Send          chan json.RawMessage
	CloseHandling context.CancelFunc
}

type Session struct {
	UserStatus string
	ActiveChat int
}

type Chat struct {
	Id      int
	UserAId uuid.UUID
	UserBId uuid.UUID
}

type Event struct {
	EventType string
	Data      json.RawMessage
}

const (
	EventSendMessageHistory string = "message_history"
	EventSetActiveChat      string = "set_active_chat"
	EventSendMessage        string = "send_message"
)

type EventSetActiveChatPayload struct {
	UID    uuid.UUID `json:"uid"`
	ChatId int       `json:"chat_id"`
}

type EventSendMessagePayload struct {
	Id          int       `json:"id"`
	ChatId      int       `json:"chat_id"`
	SenderId    uuid.UUID `json:"sender_id"`
	RecipientId uuid.UUID `json:"recipient_id"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
}
