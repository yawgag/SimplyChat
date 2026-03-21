package handler

import (
	"apiGateway/config"
	"apiGateway/internal/models"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type gatewayServiceStub struct {
	canAccessAllowed bool
	canAccessErr     error
	canAccessLogin   string
	canAccessFileID  string
}

func (g *gatewayServiceStub) Login(ctx context.Context, user *models.User) (*models.AuthTokens, error) {
	return nil, nil
}

func (g *gatewayServiceStub) Register(ctx context.Context, user *models.User) (*models.AuthTokens, error) {
	return nil, nil
}

func (g *gatewayServiceStub) Logout(ctx context.Context, refreshToken string) error {
	return nil
}

func (g *gatewayServiceStub) UpdateTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error) {
	return nil, nil
}

func (g *gatewayServiceStub) ValidateAccessToken(tokenString string) (*models.AccessToken, error) {
	return nil, nil
}

func (g *gatewayServiceStub) CanAccessFile(ctx context.Context, login string, fileID string) (bool, error) {
	g.canAccessLogin = login
	g.canAccessFileID = fileID
	return g.canAccessAllowed, g.canAccessErr
}

func (g *gatewayServiceStub) WsProxy(userConn *websocket.Conn, login string) {}

type tokensHandlerStub struct {
	token *models.AccessToken
	err   error
}

func (t *tokensHandlerStub) UpdateTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error) {
	return nil, nil
}

func (t *tokensHandlerStub) ValidateAccessToken(tokenString string) (*models.AccessToken, error) {
	return t.token, t.err
}

func TestContentFileForbiddenWhenAccessDenied(t *testing.T) {
	gateway := &gatewayServiceStub{canAccessAllowed: false}
	handler := NewMessageHandler(
		gateway,
		&tokensHandlerStub{token: &models.AccessToken{Uid: uuid.New(), Login: "alice", UserRole: "user"}},
		&config.Config{FileServiceAddr: "example.invalid"},
	)

	req := httptest.NewRequest(http.MethodGet, "/files/11111111-1111-1111-1111-111111111111/content?login=mallory", nil)
	req = mux.SetURLVars(req, map[string]string{"fileId": "11111111-1111-1111-1111-111111111111"})
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "Bearer token"})
	rec := httptest.NewRecorder()

	handler.ContentFile(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusForbidden)
	}
	if gateway.canAccessLogin != "alice" {
		t.Fatalf("expected access-check with validated login, got %q", gateway.canAccessLogin)
	}
	if gateway.canAccessFileID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected access-check for file id, got %q", gateway.canAccessFileID)
	}
}
