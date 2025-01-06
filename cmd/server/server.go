package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"log"
	"stream-recorder/internal/app/server/endpoint/restStream"
	"stream-recorder/internal/app/server/endpoint/restStreamer"
	"stream-recorder/internal/app/server/middlewares/noCache"
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
	r := gin.Default()
	r.Use(noCache.NoCacheMiddleware())

	// регистрируем эндпоинты
	serviceStreamer := restStreamer.New(a.StreamersRepo, a.ActiveM3u8, a.ActiveStreamers)
	serviceStream := restStream.New(a.ActiveM3u8, a.ActiveStreamers, a.RunnerProcess, a.Cfg)

	// регистрируем маршруты
	r.GET("/streamer/list", serviceStreamer.GetListStreamersHandler)
	r.GET("/streamer/add", serviceStreamer.AddStreamerHandler)
	r.GET("/streamer/update", serviceStreamer.UpdateStreamerHandler)
	r.GET("/streamer/delete", serviceStreamer.DeleteStreamerHandler)
	r.GET("/stream/cut", serviceStream.CutStreamHandler)
	r.GET("/stream/download_m3u8", serviceStream.DownloadM3u8Handler)

	return runServer(r, a.Cfg.Port)
}

func runServer(router *gin.Engine, port int) error {
	err := router.Run(fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	return nil
}
