package main

import (
	"apiGateway/config"
	"fmt"
	"log"
	"net/http"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("can't load config: ", err)
	}

	// server := http.Server{
	// 	Addr: cfg.GatewayAddr,
	// }
	fmt.Println("asd")
	http.ListenAndServe(cfg.GatewayAddr, nil)
	fmt.Println("asd")
}
