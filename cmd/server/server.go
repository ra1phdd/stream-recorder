package main

import (
	"log"
	"stream-recorder/internal/pkg/app"
)

func main() {
	err := app.New("server")
	if err != nil {
		log.Fatal(err)
	}
}
