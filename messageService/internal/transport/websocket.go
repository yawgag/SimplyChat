package transport

import (
	"context"
	"messageService/internal/service/messageService"
	"messageService/internal/storage/memoryStorage/clientStorage"
	"messageService/internal/storage/redis/statusStorage"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Ws struct {
	ConnStorage    clientStorage.Handler
	StatusStorage  statusStorage.Handler
	MessageService messageService.Handler
}

type Handler interface {
	InitConnection(w http.ResponseWriter, r *http.Request)
}

func NewHandler(ConnStorage clientStorage.Handler, StatusStorage statusStorage.Handler, MessageService messageService.Handler) *Ws {
	out := &Ws{
		ConnStorage:    ConnStorage,
		StatusStorage:  StatusStorage,
		MessageService: MessageService,
	}
	return out
}

var upgrader = websocket.Upgrader{ // TODO: rewrite later
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (ws *Ws) InitConnection(w http.ResponseWriter, r *http.Request) {
	// uid, err := uuid.Parse(r.Header.Get("uid"))
	// if err != nil {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }

	uid, err := uuid.Parse(r.URL.Query().Get("uid"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	ws.ConnStorage.SaveClient(uid, conn, cancel)
	ws.StatusStorage.SetOnline(ctx, uid)

	go ws.MessageService.ReadMessage(ctx, uid)
	go ws.MessageService.WriteMessage(ctx, uid)
}
