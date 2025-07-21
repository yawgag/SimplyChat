package messageService

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"messageService/internal/models"
	"messageService/internal/storage/memoryStorage/clientStorage"
	"messageService/internal/storage/postgres/messageStorage"
	"messageService/internal/storage/redis/statusStorage"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type messageHandler struct {
	ClientStorage  clientStorage.Handler
	StatusStorage  statusStorage.Handler
	MessageStorage messageStorage.Handler
}

type Handler interface {
	ReadMessage(ctx context.Context, uid uuid.UUID)
	WriteMessage(ctx context.Context, uid uuid.UUID)
}

func NewMessageHandler(clientStorage clientStorage.Handler, status statusStorage.Handler, message messageStorage.Handler) Handler {
	return &messageHandler{
		ClientStorage:  clientStorage,
		StatusStorage:  status,
		MessageStorage: message,
	}
}

func (m *messageHandler) WriteMessage(ctx context.Context, uid uuid.UUID) {
	client, err := m.ClientStorage.GetClient(uid)
	if err != nil {
		log.Println("[WriteMessage] error: ", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-client.Send:
			if !ok {
				m.ClientStorage.DeleteClient(uid)
				return
			}
			client.Conn.WriteMessage(websocket.TextMessage, msg)
		case <-time.After(time.Minute): // TODO: add heartbeat later
			continue
		}
	}
}

func (m *messageHandler) ReadMessage(ctx context.Context, uid uuid.UUID) {
	client, err := m.ClientStorage.GetClient(uid)
	if err != nil {
		log.Println("[ReadMessage] error: ", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, message, err := client.Conn.ReadMessage()
			if err != nil {
				log.Println("[ReadMessage] error: ", err)
				m.ClientStorage.DeleteClient(client.Uid)
				return
			}
			var event models.Event
			if err := json.Unmarshal(message, &event); err != nil {
				log.Println("[ReadMessage] error: ", err)
				continue
			}

			switch event.EventType {
			case models.EventSetActiveChat:
				err := m.handleNewActiveChat(event.Data)
				if err != nil {
					log.Println("[ReadMessage] error: ", err)
				}

			case models.EventSendMessage:
				err := m.handleSendMessage(event.Data)
				if err != nil {
					log.Println("[ReadMessage] error: ", err)
				}

			}
		}

	}
}

func (m *messageHandler) handleNewActiveChat(data json.RawMessage) error {
	var payload models.EventSetActiveChatPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("[handleNewActiveChat] error: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := m.StatusStorage.SetNewActiveChat(ctx, payload.UID, payload.ChatId); err != nil {
		return fmt.Errorf("[handleNewActiveChat] error: %s", err)
	}

	history, err := m.MessageStorage.GetMessageHistory(ctx, 0, payload.ChatId)
	if err != nil {
		return fmt.Errorf("[handleNewActiveChat] error: %s", err)
	}
	jsonHistory, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("[handleNewActiveChat] error: %s", err)
	}

	event := models.Event{
		EventType: models.EventSendMessageHistory,
		Data:      json.RawMessage(jsonHistory),
	}

	jsonEvent, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("[handleNewActiveChat] error: %s", err)
	}

	client, err := m.ClientStorage.GetClient(payload.UID)
	if err != nil {
		return fmt.Errorf("[handleSendMessage] error: %s", err)
	}

	client.Send <- jsonEvent // TODO: add timeout

	return nil
}

func (m *messageHandler) handleSendMessage(data json.RawMessage) error {
	var payload models.EventSendMessagePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("[handleSendMessage] error: %s", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err := m.MessageStorage.SaveMessage(ctx, &payload)
	if err != nil {
		return fmt.Errorf("[handleSendMessage] error: %s", err)
	}

	recipientStatus, err := m.StatusStorage.GetStatus(ctx, payload.RecipientId)
	if err != nil {
		return fmt.Errorf("[handleSendMessage] error: %s", err)
	}

	if recipientStatus.UserStatus != statusStorage.StatusOnline {
		//
		// TODO: add notification
		//
		return nil
	}

	recipientClient, err := m.ClientStorage.GetClient(payload.RecipientId)
	if err != nil {
		return fmt.Errorf("[handleSendMessage] error: %s", err)
	}

	recipientClient.Send <- data // TODO: add timeout
	return nil
}
