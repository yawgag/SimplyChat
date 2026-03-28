package transport

import (
	httptransport "messageService/internal/transport/http"
	"net/http/pprof"

	"github.com/gorilla/mux"
)

type Server struct {
	handler            Handler
	fileMessageHandler *httptransport.FileMessageHandler
}

func NewServer(handler Handler, fileMessageHandler *httptransport.FileMessageHandler) *Server {
	return &Server{
		handler:            handler,
		fileMessageHandler: fileMessageHandler,
	}
}

func (s *Server) InitRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/chat", s.handler.InitConnection)
	if s.fileMessageHandler != nil {
		router.HandleFunc("/chats/{chatId}/messages/files", s.fileMessageHandler.UploadFileMessage)
		router.HandleFunc("/internal/files/{fileId}/access", s.fileMessageHandler.CheckFileAccess).Methods("GET")
	}

	setupPprof(router)

	return router
}

func setupPprof(router *mux.Router) {
	// Создаем подроутер для pprof
	pprofRouter := router.PathPrefix("/debug/pprof").Subrouter()

	// Регистрируем pprof обработчики
	pprofRouter.HandleFunc("/", pprof.Index)
	pprofRouter.HandleFunc("/cmdline", pprof.Cmdline)
	pprofRouter.HandleFunc("/profile", pprof.Profile)
	pprofRouter.HandleFunc("/symbol", pprof.Symbol)
	pprofRouter.HandleFunc("/trace", pprof.Trace)

	// Добавляем обработчик для goroutine
	pprofRouter.HandleFunc("/goroutine", pprof.Handler("goroutine").ServeHTTP)
	// Добавляем обработчик для heap
	pprofRouter.HandleFunc("/heap", pprof.Handler("heap").ServeHTTP)
	// Добавляем обработчик для threadcreate
	pprofRouter.HandleFunc("/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	// Добавляем обработчик для block
	pprofRouter.HandleFunc("/block", pprof.Handler("block").ServeHTTP)
}
