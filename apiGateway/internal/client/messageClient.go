package client

import (
	"fmt"

	"github.com/gorilla/websocket"
)

type messageCilent struct {
	addr string
}

type MessageClient interface {
	ConnectToMessageService(login string) (*websocket.Conn, error)
}

func NewMessageClient(addr string) *messageCilent {
	return &messageCilent{
		addr: addr,
	}
}

func (m *messageCilent) ConnectToMessageService(login string) (*websocket.Conn, error) {
	backendURL := m.addr + "?login=" + login
	backendConn, _, err := websocket.DefaultDialer.Dial(backendURL, nil)
	if err != nil {
		return nil, fmt.Errorf("[ConnectToMessageService] error: %s", err)
	}
	return backendConn, nil
}
