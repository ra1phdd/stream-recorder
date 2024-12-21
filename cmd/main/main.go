package main

import (
	"log"
	"stream-recorder/internal/pkg/app"
)

func main() {
	err := app.New()
	if err != nil {
		log.Fatal(err)
	}
}
