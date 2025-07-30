package service

import (
	"apiGateway/internal/models"
	"context"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type GatewayService interface {
	Login(ctx context.Context, user *models.User) (*models.AuthTokens, error)
	Register(ctx context.Context, user *models.User) (*models.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error

	UpdateTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error)
	ValidateAccessToken(tokenString string) (*models.AccessToken, error)

	WsProxy(userConn *websocket.Conn, login string)
}

type gatewayService struct {
	auth          AuthService
	message       MessageService
	tokensHandler TokensHandler
}

func NewGatewayService(auth AuthService, message MessageService, tokensHandler TokensHandler) GatewayService {
	out := &gatewayService{
		auth:          auth,
		message:       message,
		tokensHandler: tokensHandler,
	}
	return out
}

func (g *gatewayService) Login(ctx context.Context, user *models.User) (*models.AuthTokens, error) {
	tokens, err := g.auth.Login(ctx, user)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func (g *gatewayService) Register(ctx context.Context, user *models.User) (*models.AuthTokens, error) {
	tokens, err := g.auth.Register(ctx, user)
	if err != nil {
		return nil, err
	}

	return tokens, err
}

func (g *gatewayService) Logout(ctx context.Context, refreshToken string) error {
	err := g.auth.Logout(ctx, refreshToken)
	return err
}

func (g *gatewayService) ValidateAccessToken(tokenString string) (*models.AccessToken, error) {
	return g.tokensHandler.ValidateAccessToken(tokenString)
}

func (g *gatewayService) UpdateTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error) {
	tokens, err := g.tokensHandler.UpdateTokens(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (g *gatewayService) WsProxy(userConn *websocket.Conn, login string) {
	defer userConn.Close()

	msgServiceConn, err := g.message.GetMsgServiceConn(login)
	if err != nil {
		log.Printf("[WsProxy] error: %s\n", err)
		return
	}
	defer msgServiceConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			msgType, msg, err := userConn.ReadMessage()
			if err != nil {
				log.Printf("[WsProxy] read from client error: %s\n", err)
				break
			}
			err = msgServiceConn.WriteMessage(msgType, msg)
			if err != nil {
				log.Printf("[WsProxy ] write to backend error: %s\n", err)
				break
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			msgType, msg, err := msgServiceConn.ReadMessage()
			if err != nil {
				log.Printf("[WsProxy] read from backend error: %s\n", err)
				break
			}
			err = userConn.WriteMessage(msgType, msg)
			if err != nil {
				log.Printf("[WsProxy] write to client error: %s\n", err)
				break
			}
		}
	}()

	wg.Wait()
}
