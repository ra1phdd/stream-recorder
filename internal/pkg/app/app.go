package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/endpoint/restStream"
	"stream-recorder/internal/app/endpoint/restStreamer"
	"stream-recorder/internal/app/middlewares/noCache"
	"stream-recorder/internal/app/repository"
	"stream-recorder/internal/app/services/m3u8"
	"stream-recorder/internal/app/services/runner"
	"stream-recorder/internal/app/services/streamlink"
	"stream-recorder/internal/app/services/streams"
	"stream-recorder/internal/app/services/tmp"
	"stream-recorder/pkg/db"
	embedded "stream-recorder/pkg/embed"
	"stream-recorder/pkg/logger"
)

type App struct {
	router        *gin.Engine
	cfg           *config.Config
	streamersRepo *repository.StreamersRepository
	runnerProcess *runner.Process
	sl            *streamlink.Streamlink
	check         *streams.Streams
}

var activeStreamers = make(map[string]bool)
var activeM3u8 = make(map[string]*m3u8.M3u8)

func New() error {
	logger.Init()

	if err := embedded.Init(); err != nil {
		return err
	}

	if err := db.Init("db/stream-recorder.db"); err != nil {
		return err
	}

	a := setupApplication()

	a.router = gin.Default()
	newRest(a)
	return runRest(a.router, a.cfg.Port)
}

func setupApplication() *App {
	var a = &App{}

	var err error
	a.cfg, err = config.New()
	if err != nil {
		logger.Fatal("Error loading config", zap.Error(err))
		return nil
	}

	logger.SetLogLevel(a.cfg.LoggerLevel)
	gin.SetMode(a.cfg.GinMode)

	err = tmp.Clear("tmp")
	if err != nil {
		logger.Error("Error clearing tmp", zap.Error(err))
		return nil
	}

	a.streamersRepo = repository.NewStreamers()
	a.runnerProcess = runner.NewProcess()

	a.sl = streamlink.New()
	a.check = streams.New(a.streamersRepo, a.sl, a.runnerProcess, a.cfg, activeStreamers, activeM3u8)

	go a.check.CheckingForStreams()
	return a
}

func newRest(a *App) {
	a.router.Use(noCache.NoCacheMiddleware())

	// регистрируем эндпоинты
	serviceStreamer := restStreamer.New(a.streamersRepo)
	serviceStream := restStream.New(activeM3u8)

	// регистрируем маршруты
	a.router.GET("/streamer/list", serviceStreamer.GetListStreamersHandler)
	a.router.GET("/streamer/add", serviceStreamer.AddStreamerHandler)
	a.router.GET("/streamer/delete", serviceStreamer.DeleteStreamerHandler)
	a.router.GET("/stream/cut", serviceStream.CutStreamHandler)
}

func runRest(router *gin.Engine, port int) error {
	err := router.Run(fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	return nil
}
