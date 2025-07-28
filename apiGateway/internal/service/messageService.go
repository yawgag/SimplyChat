package service

import (
	"apiGateway/internal/client"

	"github.com/gorilla/websocket"
)

type messageService struct {
	msg client.MessageClient
}

type MessageService interface {
	GetMsgServiceConn(login string) (*websocket.Conn, error)
}

func NewMessageService(msg client.MessageClient) *messageService {
	return &messageService{
		msg: msg,
	}
}

func (m *messageService) GetMsgServiceConn(login string) (*websocket.Conn, error) {
	return m.msg.ConnectToMessageService(login)
}
