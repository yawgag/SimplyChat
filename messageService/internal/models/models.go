package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Login         string
	Conn          *websocket.Conn
	Send          chan json.RawMessage
	CloseHandling context.CancelFunc
}

type Session struct {
	UserStatus string
	ActiveChat int
}

type Event struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

const (
	EventSendMessageHistory string = "message_history"
	EventSetActiveChat      string = "set_active_chat"
	EventSendMessage        string = "send_message"
	EventNewChat            string = "new_chat"
	EventAddUserToChat      string = "add_user_to_chat"
	EventAllUserChats       string = "all_user_chats"
)

type EventSetActiveChatPayload struct { // only [user->server] return EventSendMessageHistory
	Login  string `json:"login"`
	ChatId int    `json:"chat_id"`
}

type EventSendMessagePayload struct {
	Id          int       `json:"id"`
	ChatId      int       `json:"chat_id"`
	SenderLogin string    `json:"sender_login"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
}

type EventChatPayload struct {
	ChatId   int    `json:"chat_id"`
	ChatType string `json:"chat_type,omitempty"`
}

type EventAddUserToChatPayload struct {
	ChatId    int    `json:"chat_id"`
	UserLogin string `json:"user_login"`
}
