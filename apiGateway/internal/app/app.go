package app

import (
	"apiGateway/config"
	"apiGateway/internal/client"
	"apiGateway/internal/handler"
	"apiGateway/internal/service"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type App struct {
	Config                  *config.Config
	Router                  *mux.Router
	GRPCConn                *grpc.ClientConn // need to close connection
	AuthServicePublicRSAKey string           // need to decode jwt tokens from auth service
}

func NewApp() (*App, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("can't load config: %s", err.Error())
	}

	grpcConnection, err := grpc.NewClient(
		cfg.AuthServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("can't create grpc client: %s", err.Error())
	}

	// init auth services
	authClient := client.NewAuthClient(grpcConnection)
	authService := service.NewAuthService(authClient)

	//init message service
	messageClient := client.NewMessageClient(cfg.MessageServiceAddr)
	messageService := service.NewMessageService(messageClient)

	// get public rsa key
	publicRSAKey, err := authClient.GetPublicRSAKey(context.Background())
	if err != nil {
		return nil, err
	}

	// init gatewat
	gatewayService := service.NewGatewayService(authService, messageService)

	srv := handler.NewServer(gatewayService)

	router := srv.InitRouter()

	out := &App{
		Config:                  cfg,
		Router:                  router,
		GRPCConn:                grpcConnection,
		AuthServicePublicRSAKey: publicRSAKey,
	}
	return out, nil
}

func (a *App) Run() {
	if err := http.ListenAndServe(":8080", a.Router); err != nil { // TODO: change that addr
		log.Fatal(err)
	}
}
