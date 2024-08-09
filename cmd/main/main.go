package main

import (
	"log"
	"os"
	"os/signal"
	"stream-recorder/config"
	"stream-recorder/internal/pkg/app"
	"stream-recorder/pkg/embed"
	"stream-recorder/pkg/logger"
	"syscall"
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

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Printf("Получен сигнал: %v", sig)
		done <- true
	}()

	go func() {
		if err := application.Run(cfg.Port); err != nil {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	<-done
	if err := os.Remove(embed.GetTempFileName("streamlink")); err != nil {
		log.Printf("Ошибка удаления временного файла: %v", err)
	}
	if err := os.Remove(embed.GetTempFileName("ffmpeg")); err != nil {
		log.Printf("Ошибка удаления временного файла: %v", err)
	}
	logger.Info("Скрипт прекратил свою работу")
}
