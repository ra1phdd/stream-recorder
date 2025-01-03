package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"log"
	"stream-recorder/internal/app/endpoint/restStream"
	"stream-recorder/internal/app/endpoint/restStreamer"
	"stream-recorder/internal/app/middlewares/noCache"
	"stream-recorder/internal/app/services/tmp"
	"stream-recorder/internal/pkg/app"
	"stream-recorder/pkg/logger"
	"time"
)

func main() {
	a, err := app.New("server")
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

	err = setupServer(a)
	if err != nil {
		log.Fatal(err)
	}
}

func setupServer(a *app.App) error {
	gin.SetMode(a.Cfg.GinMode)
	a.Router = gin.Default()
	a.Router.Use(noCache.NoCacheMiddleware())

	// регистрируем эндпоинты
	serviceStreamer := restStreamer.New(a.StreamersRepo, a.ActiveM3u8, a.ActiveStreamers)
	serviceStream := restStream.New(a.ActiveM3u8, a.ActiveStreamers, a.RunnerProcess, a.Cfg)

	// регистрируем маршруты
	a.Router.GET("/streamer/list", serviceStreamer.GetListStreamersHandler)
	a.Router.GET("/streamer/add", serviceStreamer.AddStreamerHandler)
	a.Router.GET("/streamer/update", serviceStreamer.UpdateStreamerHandler)
	a.Router.GET("/streamer/delete", serviceStreamer.DeleteStreamerHandler)
	a.Router.GET("/stream/cut", serviceStream.CutStreamHandler)
	a.Router.GET("/stream/download_m3u8", serviceStream.DownloadM3u8Handler)

	return runServer(a.Router, a.Cfg.Port)
}

func runServer(router *gin.Engine, port int) error {
	err := router.Run(fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	return nil
}
