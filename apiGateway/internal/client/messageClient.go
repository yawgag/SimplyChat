package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

type messageCilent struct {
	addr       string
	httpClient *http.Client
}

type MessageClient interface {
	ConnectToMessageService(login string) (*websocket.Conn, error)
	CanAccessFile(ctx context.Context, login string, fileID string) (bool, error)
}

func NewMessageClient(addr string) *messageCilent {
	return &messageCilent{
		addr:       addr,
		httpClient: &http.Client{},
	}
}

func (m *messageCilent) ConnectToMessageService(login string) (*websocket.Conn, error) {
	backendURL := "ws://" + m.addr + "/chat?login=" + login
	backendConn, _, err := websocket.DefaultDialer.Dial(backendURL, nil)
	if err != nil {
		return nil, fmt.Errorf("[ConnectToMessageService] error: %s", err)
	}
	return backendConn, nil
}

func (m *messageCilent) CanAccessFile(ctx context.Context, login string, fileID string) (bool, error) {
	baseURL := "http://" + m.addr
	target := fmt.Sprintf("%s/internal/files/%s/access?login=%s", baseURL, url.PathEscape(fileID), url.QueryEscape(login))

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false, fmt.Errorf("[CanAccessFile] error: %s", err)
	}

	response, err := m.httpClient.Do(request)
	if err != nil {
		return false, fmt.Errorf("[CanAccessFile] error: %s", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNoContent:
		return true, nil
	case http.StatusForbidden:
		return false, nil
	default:
		return false, fmt.Errorf("[CanAccessFile] error: unexpected status %d for %s", response.StatusCode, strings.TrimSpace(fileID))
	}
}
