package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"stream-recorder/config"
	"stream-recorder/internal/app/endpoint/restStream"
	"stream-recorder/internal/app/endpoint/restStreamer"
	"stream-recorder/internal/app/middlewares/noCache"
	"stream-recorder/internal/app/services/tasks"
	"stream-recorder/pkg/logger"
)

type App struct {
	router *gin.Engine
}

func New(cfg *config.Env) (*App, error) {
	gin.SetMode(cfg.GinMode)
	logger.Init(cfg.LoggerLevel)

	err := config.UpdateJSONConfig()
	if err != nil {
		return nil, err
	}

	jsonConfig, err := config.ReadJSONConfig()
	if err != nil {
		return nil, err
	}

	for _, streamer := range jsonConfig.Streamers {
		tasks.StartTask(cfg, streamer.Username, streamer.Platform, streamer.Quality)
	}

	a := &App{}

	a.router = gin.Default()

	a.router.Use(noCache.NoCacheMiddleware())

	// регистрируем эндпоинты
	serviceStreamer := &restStreamer.Endpoint{Cfg: cfg}
	serviceStream := &restStream.Endpoint{Cfg: cfg}

	// регистрируем маршруты
	a.router.GET("/streamer/list", serviceStreamer.GetListStreamersHandler)
	a.router.GET("/streamer/add", serviceStreamer.AddStreamerHandler)
	a.router.GET("/streamer/delete", serviceStreamer.DeleteStreamerHandler)
	a.router.GET("/stream/cut", serviceStream.CutStreamHandler)

	return a, nil
}

func (a *App) Run(port string) error {
	err := a.router.Run(fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}

	return nil
}
