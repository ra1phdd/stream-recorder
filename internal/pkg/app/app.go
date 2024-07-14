package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"stream-recorder/config"
	"stream-recorder/internal/app/endpoint/restStream"
	"stream-recorder/internal/app/endpoint/restStreamer"
	"stream-recorder/internal/app/services/tasks"
	"stream-recorder/pkg/logger"
)

type App struct {
	router *gin.Engine
}

func New(cfg *config.ConfigurationEnv) (*App, error) {
	err := config.UpdateJSONConfig()
	if err != nil {
		return nil, err
	}

	gin.SetMode(cfg.GinMode)
	logger.Init(cfg.LoggerLevel)

	jsonConfig, err := config.ReadJSONConfig()
	if err != nil {
		return nil, err
	}

	for _, user := range jsonConfig.Users {
		tasks.StartTask(cfg, user.Username, user.Platform, user.Quality)
	}

	a := &App{}

	a.router = gin.Default()

	// регистрируем сервисы
	//a.streamer = streamer.New()

	// регистрируем эндпоинты
	serviceStreamer := &restStreamer.Endpoint{Cfg: cfg}
	serviceStream := &restStream.Endpoint{Cfg: cfg}

	// регистрируем маршруты
	a.router.GET("/streamer/list", serviceStreamer.GetListStreamersHandler)
	a.router.GET("/streamer/add", serviceStreamer.AddStreamerHandler)
	a.router.GET("/streamer/delete", serviceStreamer.DeleteStreamerHandler)
	//a.router.GET("/streamer/enable", serviceStreamer.GetCommandHandler)
	//a.router.GET("/streamer/disable", serviceStreamer.GetStatsHandler)
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
