package app

import (
	"context"
	"fmt"
	"log"
	"messageService/internal/config"
	"messageService/internal/service/messageService"
	"messageService/internal/storage/memoryStorage/clientStorage"
	"messageService/internal/storage/postgres"
	"messageService/internal/storage/postgres/messageStorage"
	"messageService/internal/storage/redis/statusStorage"
	"messageService/internal/transport"
	"net/http"

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
	statusStorageHandler, err := statusStorage.NewStatusStorage(context.Background(), cfg.RedisAddr)
	if err != nil {
		return nil, fmt.Errorf("[NewApp] error: %s", err)
	}

	messageServiceHandler := messageService.NewMessageHandler(clientStorageHandler, statusStorageHandler, messageStorageHandler)

	wsHandler := transport.NewHandler(clientStorageHandler, statusStorageHandler, messageServiceHandler)

	router := transport.NewServer(wsHandler).InitRouter()

	return &App{
		Config: cfg,
		Router: router,
	}, nil
}

func (a *App) Run() {
	if err := http.ListenAndServe("0.0.0.0:8081", a.Router); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Run server")
}
