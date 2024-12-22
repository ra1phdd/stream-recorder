package main

import (
	"go.uber.org/zap"
	"log"
	"stream-recorder/internal/app/services/tmp"
	"stream-recorder/internal/pkg/app"
	"stream-recorder/pkg/logger"
	"time"
)

func main() {
	a, err := app.New("client")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		if !a.Cfg.AutoCleanMediaPATH {
			return
		}
		err = tmp.ClearToTime(a.Cfg.MediaPATH, time.Duration(a.Cfg.TimeAutoCleanMediaPATH)*24*time.Hour)
		if err != nil {
			logger.Error("Error clearing directory to time", zap.Error(err))
			return
		}
	}()

	//err = setupClient(a)
	//if err != nil {
	//	log.Fatal(err)
	//}
}
