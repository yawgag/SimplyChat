package clientStorage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"messageService/internal/models"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type clientStorage struct {
	storage map[string]*models.Client
	mut     sync.RWMutex
}

func NewClientStorage() Handler {
	return &clientStorage{
		storage: make(map[string]*models.Client),
	}
}

type Handler interface {
	GetClient(login string) (*models.Client, error)
	SaveClient(login string, conn *websocket.Conn, cancel context.CancelFunc)
	DeleteClient(login string)
}

func (c *clientStorage) SaveClient(login string, conn *websocket.Conn, cancel context.CancelFunc) {
	c.mut.Lock()
	c.storage[login] = &models.Client{
		Login:         login,
		Conn:          conn,
		Send:          make(chan json.RawMessage, 32),
		CloseHandling: cancel,
	}
	c.mut.Unlock()
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(180 * time.Second))
		return nil
	})
}

func (c *clientStorage) GetClient(login string) (*models.Client, error) {
	c.mut.RLock()
	client, ok := c.storage[login]
	c.mut.RUnlock()

	if !ok {
		return nil, fmt.Errorf("user if offline")
	}
	return client, nil
}

func (c *clientStorage) DeleteClient(login string) {
	c.mut.Lock()
	defer c.mut.Unlock()

	client, ok := c.storage[login]
	if !ok {
		return
	}

	client.CloseHandling()
	err := client.Conn.Close()
	if err != nil {
		log.Println("[DeleteClient] error: ", err)
	}
	close(client.Send)
	delete(c.storage, login)

}
