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
	EventSendMessageHistory  string = "message_history"
	EventMessageHistoryError string = "message_history_error"
	EventGetMessageHistory   string = "get_message_history"
	EventSetActiveChat       string = "set_active_chat"
	EventSendMessage         string = "send_message"
	EventNewChat             string = "new_chat"
	EventAddUserToChat       string = "add_user_to_chat"
	EventAllUserChats        string = "all_user_chats"
)

type EventSetActiveChatPayload struct { // only [user->server] return EventSendMessageHistory
	Login  string `json:"login"`
	ChatId int    `json:"chat_id"`
}

type MessageHistoryCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        int       `json:"id"`
}

type EventGetMessageHistoryPayload struct {
	Login  string                `json:"login"`
	ChatId int                   `json:"chat_id"`
	Limit  int                   `json:"limit,omitempty"`
	Before *MessageHistoryCursor `json:"before,omitempty"`
}

type EventSendMessageHistoryPayload struct {
	ChatId     int                       `json:"chat_id"`
	Items      []EventSendMessagePayload `json:"items"`
	HasMore    bool                      `json:"has_more"`
	NextCursor *MessageHistoryCursor     `json:"next_cursor,omitempty"`
}

type EventMessageHistoryErrorPayload struct {
	ChatId  int    `json:"chat_id"`
	Message string `json:"message"`
}

type EventSendMessagePayload struct {
	Id          int                 `json:"id"`
	ChatId      int                 `json:"chat_id"`
	SenderLogin string              `json:"sender_login"`
	Kind        string              `json:"kind"`
	Content     string              `json:"content"`
	Attachments []AttachmentPayload `json:"attachments"`
	CreatedAt   time.Time           `json:"created_at"`
}

type AttachmentPayload struct {
	FileID           string `json:"file_id"`
	OriginalFilename string `json:"original_filename"`
	MimeType         string `json:"mime_type"`
	Size             int64  `json:"size"`
	Kind             string `json:"kind"`
	ContentURL       string `json:"content_url"`
	DownloadURL      string `json:"download_url"`
}

type EventChatPayload struct {
	ChatId   int    `json:"chat_id"`
	ChatType string `json:"chat_type,omitempty"`
}

type EventAddUserToChatPayload struct {
	ChatId    int    `json:"chat_id"`
	UserLogin string `json:"user_login"`
}
