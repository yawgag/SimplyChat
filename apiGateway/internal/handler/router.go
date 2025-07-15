package handler

import (
	"apiGateway/internal/service"

	"github.com/gorilla/mux"
)

type Server struct {
	Gateway service.GatewayService
}

func NewServer(gateway service.GatewayService) *Server {
	return &Server{Gateway: gateway}
}

func (s *Server) InitRouter() *mux.Router {
	router := mux.NewRouter()

	authHandler := NewAuthHandler(s.Gateway)
	router.HandleFunc("/login", authHandler.Login)
	router.HandleFunc("/register", authHandler.Register)
	router.HandleFunc("/logout", authHandler.Register)

	return router
}
