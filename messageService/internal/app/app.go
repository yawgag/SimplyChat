package app

import (
	"context"
	"fmt"
	"log"
	fileServiceClient "messageService/internal/client/fileService"
	"messageService/internal/config"
	"messageService/internal/service/messageService"
	"messageService/internal/storage/memoryStorage/clientStorage"
	"messageService/internal/storage/postgres"
	"messageService/internal/storage/postgres/messageStorage"
	"messageService/internal/storage/redis/chatMembersStorage"
	"messageService/internal/transport"
	httptransport "messageService/internal/transport/http"
	"net/http"

	_ "net/http/pprof"

	"github.com/gorilla/mux"
)

type App struct {
	Config *config.Config
	Router *mux.Router
}

func NewApp() (*App, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("[NewApp] error: %s", err)
	}

	dbConnPool, err := postgres.InitDb(cfg.DbURL)
	if err != nil {
		return nil, fmt.Errorf("[NewApp] error: %s", err)
	}

	clientStorageHandler := clientStorage.NewClientStorage()
	messageStorageHandler := messageStorage.NewMessageStorage(dbConnPool)
	chatMembersStorage, err := chatMembersStorage.NewChatMemberStorage(context.Background(), cfg.RedisAddr)
	if err != nil {
		return nil, fmt.Errorf("[NewApp] error: %s", err)
	}
	// statusStorageHandler, err := statusStorage.NewStatusStorage(context.Background(), cfg.RedisAddr)

	fileClient := fileServiceClient.New(cfg.FileServiceURL, cfg.FileServiceUploadTimeout)
	messageServiceHandler := messageService.NewMessageHandler(clientStorageHandler, chatMembersStorage, messageStorageHandler, fileClient)

	wsHandler := transport.NewHandler(clientStorageHandler, messageServiceHandler)
	fileMessageHandler := httptransport.NewFileMessageHandler(messageServiceHandler)

	router := transport.NewServer(wsHandler, fileMessageHandler).InitRouter()

	return &App{
		Config: cfg,
		Router: router,
	}, nil
}

func (a *App) Run() {
	if err := http.ListenAndServe(a.Config.ServiceAddr, a.Router); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Run server")
}
