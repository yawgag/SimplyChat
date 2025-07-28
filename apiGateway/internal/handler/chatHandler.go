package handler

import (
	"apiGateway/internal/service"
	"net/http"

	"github.com/gorilla/websocket"
)

type messageHandler struct {
	gateway       service.GatewayService
	tokensHandler service.TokensHandler
}

func NewMessageHandler(gateway service.GatewayService, tokensHandler service.TokensHandler) *messageHandler {
	return &messageHandler{
		gateway:       gateway,
		tokensHandler: tokensHandler,
	}

}

var upgrader = websocket.Upgrader{ // TODO: rewrite later
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (m *messageHandler) ConnectToChat(w http.ResponseWriter, r *http.Request) {
	accessTokenCookie, err := r.Cookie("access_token")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	_, err = m.tokensHandler.ValidateAccessToken(accessTokenCookie.Value)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	login := r.URL.Query().Get("login")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go m.gateway.WsProxy(conn, login)

}
