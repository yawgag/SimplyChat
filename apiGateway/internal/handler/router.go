package handler

import (
	"apiGateway/internal/service"
	"net/http"

	"github.com/gorilla/mux"
)

type Server struct {
	Gateway       service.GatewayService
	tokensHandler service.TokensHandler
}

func NewServer(gateway service.GatewayService) *Server {
	return &Server{Gateway: gateway}
}

func (s *Server) InitRouter() *mux.Router {
	router := mux.NewRouter()

	authHandler := NewAuthHandler(s.Gateway)
	router.HandleFunc("/login", authHandler.Login)
	router.HandleFunc("/register", authHandler.Register)
	router.HandleFunc("/logout", authHandler.Logout)

	messageHandler := NewMessageHandler(s.Gateway, s.tokensHandler)
	router.HandleFunc("/chat", messageHandler.ConnectToChat)

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
