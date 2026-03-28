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
	defaultHistoryPageLimit   int    = 50
	maxHistoryPageLimit       int    = 100
)

type messageHandler struct {
	ClientStorage      clientStorage.Handler
	ChatMembersStorage chatMembersStorage.Handler
	MessageStorage     messageStorage.Handler
	FileService        FileServiceClient
}

type Handler interface {
	ReadMessage(ctx context.Context, login string)
	WriteMessage(ctx context.Context, login string)
	SendFileMessage(ctx context.Context, input SendFileMessageInput) (*models.EventSendMessagePayload, error)
	CanAccessFile(ctx context.Context, login string, fileID string) (bool, error)
}

func NewMessageHandler(clientStorage clientStorage.Handler, chatMembersStorage chatMembersStorage.Handler, message messageStorage.Handler, fileService FileServiceClient) Handler {
	return &messageHandler{
		ClientStorage:      clientStorage,
		ChatMembersStorage: chatMembersStorage,
		MessageStorage:     message,
		FileService:        fileService,
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
			case models.EventGetMessageHistory:
				err := m.handleGetMessageHistory(event.Data, login)
				if err != nil {
					log.Println("[EventGetMessageHistory] error: ", err)
				}
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

	if payload.Login == "" || payload.ChatId <= 0 {
		return fmt.Errorf("[handleNewActiveChat] error: bad request")
	}

	return nil
}

func (m *messageHandler) handleGetMessageHistory(data json.RawMessage, login string) error {
	var payload models.EventGetMessageHistoryPayload
	fail := func(chatID int, message string, err error) error {
		if sendErr := m.sendHistoryError(login, chatID, message); sendErr != nil {
			log.Println("[sendHistoryError] error: ", sendErr)
		}
		if err != nil {
			return fmt.Errorf("[handleGetMessageHistory] error: %w", err)
		}
		return fmt.Errorf("[handleGetMessageHistory] error: %s", message)
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return fail(0, "invalid history request payload", err)
	}
	if payload.ChatId <= 0 {
		return fail(payload.ChatId, "invalid chat id", nil)
	}
	if payload.Login != "" && payload.Login != login {
		return fail(payload.ChatId, "login mismatch", nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	exists, err := m.MessageStorage.ChatExists(ctx, payload.ChatId)
	if err != nil {
		return fail(payload.ChatId, "failed to check chat", err)
	}
	if !exists {
		return fail(payload.ChatId, "chat doesn't exist", nil)
	}

	isMember, err := m.MessageStorage.IsChatMember(ctx, payload.ChatId, login)
	if err != nil {
		return fail(payload.ChatId, "failed to check chat membership", err)
	}
	if !isMember {
		return fail(payload.ChatId, "access denied", nil)
	}

	limit := payload.Limit
	if limit <= 0 {
		limit = defaultHistoryPageLimit
	}
	if limit > maxHistoryPageLimit {
		limit = maxHistoryPageLimit
	}

	page, err := m.MessageStorage.GetMessageHistory(ctx, messageStorage.HistoryPageParams{
		ChatID: payload.ChatId,
		Limit:  limit,
		Before: payload.Before,
	})
	if err != nil {
		return fail(payload.ChatId, "failed to load message history", err)
	}
	for i := range page.Items {
		enrichMessagePayload(&page.Items[i])
	}

	jsonHistory, err := json.Marshal(page)
	if err != nil {
		return fail(payload.ChatId, "failed to encode message history", err)
	}

	event := models.Event{
		EventType: models.EventSendMessageHistory,
		Data:      json.RawMessage(jsonHistory),
	}

	jsonEvent, err := json.Marshal(event)
	if err != nil {
		return fail(payload.ChatId, "failed to encode history event", err)
	}

	client, err := m.ClientStorage.GetClient(login)
	if err != nil {
		return fail(payload.ChatId, "client is offline", err)
	}

	select {
	case client.Send <- jsonEvent:
		return nil
	case <-time.After(time.Second * 3):
		return fail(payload.ChatId, "history response timeout", nil)
	}
}

func (m *messageHandler) sendHistoryError(login string, chatID int, message string) error {
	client, err := m.ClientStorage.GetClient(login)
	if err != nil {
		return err
	}

	data, err := json.Marshal(models.EventMessageHistoryErrorPayload{
		ChatId:  chatID,
		Message: message,
	})
	if err != nil {
		return err
	}

	payload, err := json.Marshal(models.Event{
		EventType: models.EventMessageHistoryError,
		Data:      data,
	})
	if err != nil {
		return err
	}

	select {
	case client.Send <- payload:
		return nil
	case <-time.After(time.Second * 3):
		return fmt.Errorf("send history error timeout")
	}
}

func (m *messageHandler) handleSendMessage(data json.RawMessage) error {
	var payload models.EventSendMessagePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("[handleSendMessage] error: %s", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	savedMessage, err := m.MessageStorage.SaveMessage(ctx, &payload)
	if err != nil {
		return fmt.Errorf("[handleSendMessage] error: %s", err)
	}
	enrichMessagePayload(savedMessage)

	chatMembersLogins, err := m.getChatMembers(ctx, payload.ChatId)
	if err != nil {
		return fmt.Errorf("[handleSendMessage] error: %s", err)
	}

	jsonEventMessage, err := json.Marshal(models.Event{
		EventType: models.EventSendMessage,
		Data:      mustMarshalRaw(savedMessage),
	})
	if err != nil {
		return fmt.Errorf("[handleSendMessage] error: %s", err)
	}

	for _, userLogin := range chatMembersLogins {
		if userLogin == savedMessage.SenderLogin {
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
			continue
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

func (m *messageHandler) fanOutMessage(ctx context.Context, payload *models.EventSendMessagePayload) error {
	chatMembersLogins, err := m.getChatMembers(ctx, payload.ChatId)
	if err != nil {
		return fmt.Errorf("[fanOutMessage] error: %s", err)
	}

	jsonEventMessage, err := json.Marshal(models.Event{
		EventType: models.EventSendMessage,
		Data:      mustMarshalRaw(payload),
	})
	if err != nil {
		return fmt.Errorf("[fanOutMessage] error: %s", err)
	}

	for _, userLogin := range chatMembersLogins {
		if userLogin == payload.SenderLogin {
			continue
		}

		client, err := m.ClientStorage.GetClient(userLogin)
		if err != nil {
			continue
		}

		select {
		case client.Send <- jsonEventMessage:
		case <-time.After(time.Second * 3):
			return fmt.Errorf("[fanOutMessage] error: 3 second timeout")
		}
	}

	return nil
}

func enrichMessagePayload(payload *models.EventSendMessagePayload) {
	if payload.Kind == "" {
		payload.Kind = messageKindText
	}
	if payload.Kind == messageKindFile {
		payload.Content = ""
	}
	if payload.Attachments == nil {
		payload.Attachments = []models.AttachmentPayload{}
	}
	for i := range payload.Attachments {
		payload.Attachments[i].ContentURL = buildContentURL(payload.Attachments[i].FileID)
		payload.Attachments[i].DownloadURL = buildDownloadURL(payload.Attachments[i].FileID)
	}
}

func buildContentURL(fileID string) string {
	return "/files/" + fileID + "/content"
}

func buildDownloadURL(fileID string) string {
	return "/files/" + fileID + "/download"
}

func mustMarshalRaw(payload *models.EventSendMessagePayload) json.RawMessage {
	data, _ := json.Marshal(payload)
	return data
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

func (m *messageHandler) CanAccessFile(ctx context.Context, login string, fileID string) (bool, error) {
	return m.MessageStorage.UserHasAccessToFile(ctx, login, fileID)
}
