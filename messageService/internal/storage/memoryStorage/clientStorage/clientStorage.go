package clientStorage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"messageService/internal/models"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type clientStorage struct {
	storage map[uuid.UUID]*models.Client
	mut     sync.RWMutex
}

func NewClientStorage() Handler {
	return &clientStorage{
		storage: make(map[uuid.UUID]*models.Client),
	}
}

type Handler interface {
	GetClient(uid uuid.UUID) (*models.Client, error)
	SaveClient(uid uuid.UUID, conn *websocket.Conn, cancel context.CancelFunc)
	DeleteClient(uid uuid.UUID)
}

func (c *clientStorage) SaveClient(uid uuid.UUID, conn *websocket.Conn, cancel context.CancelFunc) {
	c.mut.Lock()
	c.storage[uid] = &models.Client{
		Uid:           uid,
		Conn:          conn,
		Send:          make(chan json.RawMessage, 32),
		CloseHandling: cancel,
	}
	c.mut.Unlock()
}

func (c *clientStorage) GetClient(uid uuid.UUID) (*models.Client, error) {
	c.mut.RLock()
	client, ok := c.storage[uid]
	c.mut.RUnlock()
	if !ok {
		return nil, fmt.Errorf("connection doesn't exist")
	}
	return client, nil
}

func (c *clientStorage) DeleteClient(uid uuid.UUID) {
	c.mut.Lock()
	defer c.mut.Unlock()

	client, ok := c.storage[uid]
	if !ok {
		return
	}

	client.CloseHandling()
	err := client.Conn.Close()
	if err != nil {
		log.Println("[DeleteClient] error: ", err)
	}
	close(client.Send)
	delete(c.storage, uid)

}
