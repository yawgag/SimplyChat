package handler

import (
	"apiGateway/config"
	"apiGateway/internal/service"
	"net/http"

	"github.com/gorilla/mux"
)

type Server struct {
	Gateway       service.GatewayService
	tokensHandler service.TokensHandler
	config        *config.Config
}

func NewServer(gateway service.GatewayService, tokensHandler service.TokensHandler, cfg *config.Config) *Server {
	return &Server{
		Gateway:       gateway,
		tokensHandler: tokensHandler,
		config:        cfg,
	}
}

func (s *Server) InitRouter() *mux.Router {
	router := mux.NewRouter()

	messageHandler := NewMessageHandler(s.Gateway, s.tokensHandler, s.config)
	router.HandleFunc("/chat", messageHandler.ConnectToChat)
	router.HandleFunc("/chats/{chatId}/messages/files", messageHandler.UploadFileMessage).Methods(http.MethodPost)
	router.HandleFunc("/files/{fileId}/content", messageHandler.ContentFile).Methods(http.MethodGet)
	router.HandleFunc("/files/{fileId}/download", messageHandler.DownloadFile).Methods(http.MethodGet)

	authHandler := NewAuthHandler(s.Gateway)
	router.HandleFunc("/api/auth/refresh", authHandler.HandleUpdateTokens)
	router.HandleFunc("/api/auth/check", authHandler.HandleCheckAuth)
	router.HandleFunc("/login", authHandler.Login)
	router.HandleFunc("/register", authHandler.Register)
	router.HandleFunc("/logout", authHandler.Logout)

	// TODO: rewrite part below
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	router.HandleFunc("/auth.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/auth.html")
	})

	fs := http.FileServer(http.Dir("static/"))
	router.PathPrefix("/").Handler(http.StripPrefix("/", fs))

	return router
}
