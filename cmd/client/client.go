package main

import (
	"log"
	"stream-recorder/internal/pkg/app"
)

func main() {
	err := app.New("client")
	if err != nil {
		log.Fatal(err)
	}

	//err = setupClient(a)
	//if err != nil {
	//	log.Fatal(err)
	//}
}

//func setupClient(a *app.App) error {
//	wails := application.New(application.Options{
//		Name:        "stream-recorder",
//		Description: "A demo of using raw HTML & CSS",
//		Services: []application.Service{
//			application.NewService(a.s),
//		},
//		Assets: application.AssetOptions{
//			Handler: application.AssetFileServerFS(assets.Get()),
//		},
//		Mac: application.MacOptions{
//			ApplicationShouldTerminateAfterLastWindowClosed: true,
//		},
//	})
//
//	wails.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
//		Title: "stream-recorder",
//		Mac: application.MacWindow{
//			InvisibleTitleBarHeight: 50,
//			Backdrop:                application.MacBackdropTranslucent,
//			TitleBar:                application.MacTitleBarHiddenInset,
//		},
//		BackgroundColour: application.NewRGB(27, 38, 54),
//		URL:              "/",
//	})
//
//	go func() {
//		for {
//			now := time.Now().Format(time.RFC1123)
//			wails.EmitEvent("time", now)
//			time.Sleep(time.Second)
//		}
//	}()
//
//	return wails.Run()
//}
