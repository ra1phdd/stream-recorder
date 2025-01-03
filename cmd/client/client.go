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
		for {
			err := tmp.RemoveEmptyDirs(a.Cfg.TempPATH)
			if err != nil {
				logger.Error("Error clearing empty directory", zap.Error(err))
				return
			}
			time.Sleep(3 * time.Hour)
		}
	}()

	go func() {
		for {
			if !a.Cfg.AutoCleanMediaPATH {
				break
			}
			err = tmp.ClearToTime(a.Cfg.MediaPATH, time.Duration(a.Cfg.TimeAutoCleanMediaPATH)*24*time.Hour)
			if err != nil {
				logger.Error("Error clearing directory to time", zap.Error(err))
				return
			}
			time.Sleep(3 * time.Hour)
		}
	}()

	//err = setupClient(a)
	//if err != nil {
	//	log.Fatal(err)
	//}
}
