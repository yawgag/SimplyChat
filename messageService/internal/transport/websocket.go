package transport

import (
	"context"
	"messageService/internal/service/messageService"
	"messageService/internal/storage/memoryStorage/clientStorage"
	"net/http"

	"github.com/gorilla/websocket"
)

type Ws struct {
	ConnStorage    clientStorage.Handler
	MessageService messageService.Handler
}

type Handler interface {
	InitConnection(w http.ResponseWriter, r *http.Request)
}

func NewHandler(ConnStorage clientStorage.Handler, MessageService messageService.Handler) *Ws {
	out := &Ws{
		ConnStorage:    ConnStorage,
		MessageService: MessageService,
	}
	return out
}

var upgrader = websocket.Upgrader{ // TODO: rewrite later
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (ws *Ws) InitConnection(w http.ResponseWriter, r *http.Request) {
	login := r.URL.Query().Get("login")
	if login == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	ws.ConnStorage.SaveClient(login, conn, cancel)

	go ws.MessageService.ReadMessage(ctx, login)
	go ws.MessageService.WriteMessage(ctx, login)
}
