package main

import (
	"log"
	"stream-recorder/config"
	"stream-recorder/internal/pkg/app"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("%+v\n", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = application.Run(cfg.Port)
	if err != nil {
		log.Fatal(err)
	}
}
