package service

import (
	"apiGateway/internal/client"
	"context"

	"github.com/gorilla/websocket"
)

type messageService struct {
	msg client.MessageClient
}

type MessageService interface {
	GetMsgServiceConn(login string) (*websocket.Conn, error)
	CanAccessFile(ctx context.Context, login string, fileID string) (bool, error)
}

func NewMessageService(msg client.MessageClient) *messageService {
	return &messageService{
		msg: msg,
	}
}

func (m *messageService) GetMsgServiceConn(login string) (*websocket.Conn, error) {
	return m.msg.ConnectToMessageService(login)
}

func (m *messageService) CanAccessFile(ctx context.Context, login string, fileID string) (bool, error) {
	return m.msg.CanAccessFile(ctx, login, fileID)
}
