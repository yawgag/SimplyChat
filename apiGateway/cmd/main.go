package main

import (
	"apiGateway/internal/app"
	"log"
)

func main() {
	app, err := app.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.Run()
}
