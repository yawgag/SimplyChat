package messageService

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"messageService/internal/models"
	"messageService/internal/storage/memoryStorage/clientStorage"
	"messageService/internal/storage/postgres/messageStorage"
	"messageService/internal/storage/redis/chatMembersStorage"
	"time"

	"github.com/gorilla/websocket"
)

const (
	chatTypePublic            string = "public"
	chatTypePrivate           string = "private"
	numberOfUserInPrivateChat int    = 2
	numberOfUserInPublicChat  int    = 250
)

type messageHandler struct {
	ClientStorage      clientStorage.Handler
	ChatMembersStorage chatMembersStorage.Handler
	MessageStorage     messageStorage.Handler
}

type Handler interface {
	ReadMessage(ctx context.Context, login string)
	WriteMessage(ctx context.Context, login string)
}

func NewMessageHandler(clientStorage clientStorage.Handler, chatMembersStorage chatMembersStorage.Handler, message messageStorage.Handler) Handler {
	return &messageHandler{
		ClientStorage:      clientStorage,
		ChatMembersStorage: chatMembersStorage,
		MessageStorage:     message,
	}
}

func (m *messageHandler) WriteMessage(ctx context.Context, login string) {
	client, err := m.ClientStorage.GetClient(login)
	if err != nil {
		log.Println("[WriteMessage] error: ", err)
		return
	}

	heartbeatTicker := time.NewTicker(60 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-client.Send:
			if !ok {
				m.ClientStorage.DeleteClient(login)
				return
			}
			client.Conn.WriteMessage(websocket.TextMessage, msg)
		case <-heartbeatTicker.C:
			err := client.Conn.WriteControl(
				websocket.PingMessage,
				[]byte("heartbeat"),
				time.Now().Add(60*time.Second),
			)
			if err != nil {
				log.Printf("[WriteMessage] heartbeat error for client %s: %v\n", login, err)
				m.ClientStorage.DeleteClient(login)
				return
			}

		}
	}
}

func (m *messageHandler) ReadMessage(ctx context.Context, login string) {
	client, err := m.ClientStorage.GetClient(login)
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
				m.ClientStorage.DeleteClient(client.Login)
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
					log.Println("[EventSetActiveChat] error: ", err)
				}

			case models.EventSendMessage:
				err := m.handleSendMessage(event.Data)
				if err != nil {
					log.Println("[EventSendMessage] error: ", err)
				}
			case models.EventAddUserToChat:
				err := m.handleNewUserInChat(event.Data)
				if err != nil {
					log.Println("[EventAddUserToChat] error: ", err)
				}
			case models.EventNewChat:
				err := m.handleNewChat(event.Data, login)
				if err != nil {
					log.Println("[EventNewChat] error: ", err)
				}
			case models.EventAllUserChats:
				err := m.handleAllUserChats(login)
				if err != nil {
					log.Println("[EventAllUserChats] error: ", err)
				}
			}
		}

	}
}

func (m *messageHandler) handleNewChat(data json.RawMessage, login string) error {
	var payload models.EventChatPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("[handleNewChat] error: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var newChatId int

	switch payload.ChatType {
	case chatTypePublic:
		chatId, err := m.MessageStorage.CreateChat(ctx, numberOfUserInPublicChat)
		if err != nil {
			return fmt.Errorf("[handleNewChat] error: %s", err)
		}
		newChatId = chatId
	case chatTypePrivate:
		chatId, err := m.MessageStorage.CreateChat(ctx, numberOfUserInPrivateChat)
		if err != nil {
			return fmt.Errorf("[handleNewChat] error: %s", err)
		}
		newChatId = chatId
	default:
		return fmt.Errorf("[handleNewChat] error: bad request")
	}

	err := m.MessageStorage.AddUserToChat(ctx, newChatId, login)
	if err != nil {
		return fmt.Errorf("[handleNewChat] error: %s", err)
	}

	// maybe user DRY in code below
	outData, err := json.Marshal(models.EventAddUserToChatPayload{
		ChatId:    newChatId,
		UserLogin: login,
	})
	if err != nil {
		return fmt.Errorf("[handleNewChat] error: %s", err)
	}

	event := models.Event{
		EventType: models.EventAddUserToChat,
		Data:      outData,
	}

	jsonEvent, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("[handleNewChat] error: %s", err)
	}

	client, err := m.ClientStorage.GetClient(login)
	if err != nil {
		return fmt.Errorf("[handleNewChat] error: %s", err)
	}

	select {
	case client.Send <- jsonEvent:
		return nil
	case <-time.After(time.Second * 3):
		return fmt.Errorf("[handleNewChat] error: 3 second timeout")
	}
}

func (m *messageHandler) handleNewUserInChat(data json.RawMessage) error {
	var payload models.EventAddUserToChatPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("[handleNewUserInChat] error: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := m.MessageStorage.AddUserToChat(ctx, payload.ChatId, payload.UserLogin)
	if err != nil {
		return fmt.Errorf("[handleNewUserInChat] error: %s", err)
	}

	outData, err := json.Marshal(models.EventAddUserToChatPayload{
		ChatId:    payload.ChatId,
		UserLogin: payload.UserLogin,
	})
	if err != nil {
		return fmt.Errorf("[handleNewUserInChat] error: %s", err)
	}

	event := models.Event{
		EventType: models.EventAddUserToChat,
		Data:      outData,
	}

	jsonEvent, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("[handleNewUserInChat] error: %s", err)
	}

	client, err := m.ClientStorage.GetClient(payload.UserLogin)
	if err != nil {
		//
		// user is offline. Send notification
		//
		return fmt.Errorf("[handleNewUserInChat] error: %s", err)
	}

	select {
	case client.Send <- jsonEvent:
		return nil
	case <-time.After(time.Second * 3):
		return fmt.Errorf("[handleNewUserInChat] error: 3 second timeout")
	}
}

func (m *messageHandler) handleNewActiveChat(data json.RawMessage) error {
	var payload models.EventSetActiveChatPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("[handleNewActiveChat] error: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

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

	client, err := m.ClientStorage.GetClient(payload.Login)
	if err != nil {
		//
		// user is offline. Send notification
		//
		return fmt.Errorf("[handleNewActiveChat] error: %s", err)
	}

	select {
	case client.Send <- jsonEvent:
		return nil
	case <-time.After(time.Second * 3):
		return fmt.Errorf("[handleNewActiveChat] error: 3 second timeout")
	}
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

	chatMembersLogins, err := m.getChatMembers(ctx, payload.ChatId)
	if err != nil {
		return fmt.Errorf("[handleSendMessage] error: %s", err)
	}

	jsonEventMessage, err := json.Marshal(models.Event{
		EventType: models.EventSendMessage,
		Data:      data,
	})
	if err != nil {
		return fmt.Errorf("[handleSendMessage] error: %s", err)
	}

	for _, userLogin := range chatMembersLogins {
		if userLogin == payload.SenderLogin {
			continue
		}

		client, err := m.ClientStorage.GetClient(userLogin)
		if err != nil {
			//
			// user is offline. Send notification
			//
			continue
		}

		select {
		case client.Send <- jsonEventMessage:
			return nil
		case <-time.After(time.Second * 3):
			return fmt.Errorf("[handleSendMessage] error: 3 second timeout")
		}
	}

	return nil
}

func (m *messageHandler) getChatMembers(ctx context.Context, chatId int) ([]string, error) {
	members, err := m.ChatMembersStorage.GetMembersList(ctx, chatId)
	if err != nil {
		log.Println("[GetChatMembers] redis error: ", err)
	}

	if len(members) == 0 {
		members, err = m.MessageStorage.GetAllChatMembers(ctx, chatId)
		if err != nil {
			return nil, fmt.Errorf("[GetChatMembers] redis error: %s", err)
		}
	}

	err = m.ChatMembersStorage.CreateMembersList(ctx, chatId, members...)
	if err != nil {
		log.Println("[GetChatMembers] redis error: ", err)
	}

	return members, nil
}

func (m *messageHandler) handleAllUserChats(login string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	chats, err := m.MessageStorage.GetAllUserChats(ctx, login)

	if err != nil {
		return fmt.Errorf("[handleAllUserChats] error: %s", err)
	}

	jsonChats, err := json.Marshal(chats)
	if err != nil {
		return fmt.Errorf("[handleAllUserChats] error: %s", err)
	}

	event := &models.Event{
		EventType: models.EventAllUserChats,
		Data:      jsonChats,
	}

	jsonEvent, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("[handleAllUserChats] error: %s", err)
	}

	client, err := m.ClientStorage.GetClient(login)
	if err != nil {
		return fmt.Errorf("[handleAllUserChats] error: %s", err)
	}

	client.Send <- jsonEvent

	return nil
}
