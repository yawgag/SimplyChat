package handler

import (
	"apiGateway/config"
	"apiGateway/internal/models"
	"apiGateway/internal/service"
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type messageHandler struct {
	gateway       service.GatewayService
	tokensHandler service.TokensHandler
	config        *config.Config
}

func NewMessageHandler(gateway service.GatewayService, tokensHandler service.TokensHandler, cfg *config.Config) *messageHandler {
	return &messageHandler{
		gateway:       gateway,
		tokensHandler: tokensHandler,
		config:        cfg,
	}

}

var upgrader = websocket.Upgrader{ // TODO: rewrite later
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (m *messageHandler) ConnectToChat(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.getValidatedAccessToken(w, r); !ok {
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

func (m *messageHandler) UploadFileMessage(w http.ResponseWriter, r *http.Request) {
	token, ok := m.getValidatedAccessToken(w, r)
	if !ok {
		return
	}
	request := r.Clone(r.Context())
	query := request.URL.Query()
	query.Set("login", token.Login)
	request.URL.RawQuery = query.Encode()

	log.Printf("[apiGateway][UploadFileMessage] method=%s path=%s login=%s target=%s", request.Method, request.URL.Path, token.Login, m.config.MessageServiceAddr)
	m.proxyRequest(w, request, "http://"+m.config.MessageServiceAddr)
}

func (m *messageHandler) ContentFile(w http.ResponseWriter, r *http.Request) {
	token, ok := m.getValidatedAccessToken(w, r)
	if !ok {
		return
	}
	if !m.ensureFileAccess(w, r, token.Login) {
		return
	}
	log.Printf("[apiGateway][ContentFile] method=%s path=%s target=%s", r.Method, r.URL.Path, m.config.FileServiceAddr)
	m.proxyRequest(w, r, "http://"+m.config.FileServiceAddr)
}

func (m *messageHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	token, ok := m.getValidatedAccessToken(w, r)
	if !ok {
		return
	}
	if !m.ensureFileAccess(w, r, token.Login) {
		return
	}
	log.Printf("[apiGateway][DownloadFile] method=%s path=%s target=%s", r.Method, r.URL.Path, m.config.FileServiceAddr)
	m.proxyRequest(w, r, "http://"+m.config.FileServiceAddr)
}

func (m *messageHandler) getValidatedAccessToken(w http.ResponseWriter, r *http.Request) (*models.AccessToken, bool) {
	accessTokenCookie, err := r.Cookie("access_token")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, false
	}

	token, err := m.tokensHandler.ValidateAccessToken(accessTokenCookie.Value)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, false
	}

	return token, true
}

func (m *messageHandler) ensureFileAccess(w http.ResponseWriter, r *http.Request, login string) bool {
	fileID := mux.Vars(r)["fileId"]
	if fileID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	allowed, err := m.gateway.CanAccessFile(ctx, login, fileID)
	if err != nil {
		log.Printf("[apiGateway][ensureFileAccess] login=%s file_id=%s err=%v", login, fileID, err)
		w.WriteHeader(http.StatusBadGateway)
		return false
	}
	if !allowed {
		w.WriteHeader(http.StatusForbidden)
		return false
	}

	return true
}

func (m *messageHandler) proxyRequest(w http.ResponseWriter, r *http.Request, target string) {
	targetURL, err := url.Parse(target)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = r.URL.Path
		req.URL.RawQuery = r.URL.RawQuery
		req.Host = targetURL.Host
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		log.Printf("[apiGateway][proxyRequest] upstream=%s method=%s path=%s status=%d", targetURL.String(), r.Method, r.URL.Path, resp.StatusCode)
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[apiGateway][proxyRequest] upstream=%s method=%s path=%s error=%v", targetURL.String(), r.Method, r.URL.Path, err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}
