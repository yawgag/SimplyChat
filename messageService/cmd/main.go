package main

import (
	"log"
	"messageService/internal/app"
)

func main() {
	app, err := app.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.Run()
}
