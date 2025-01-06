package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"
	"log"
	"stream-recorder/internal/app/client"
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

	err = setupClient(a)
	if err != nil {
		log.Fatal(err)
	}
}

func setupClient(a *app.App) error {
	wails := application.New(application.Options{
		Name:        "stream-recorder",
		Description: "A demo of using raw HTML & CSS",
		Services: []application.Service{
			application.NewService(a.CheckStreams),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets.Get()),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	wails.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title: "stream-recorder",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			wails.EmitEvent("time", now)
			time.Sleep(time.Second)
		}
	}()

	return wails.Run()
}
