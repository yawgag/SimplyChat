package transport

import "github.com/gorilla/mux"

type Server struct {
	handler Handler
}

func NewServer(handler Handler) *Server {
	return &Server{handler: handler}
}

func (s *Server) InitRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/chat", s.handler.InitConnection)

	return router
}
