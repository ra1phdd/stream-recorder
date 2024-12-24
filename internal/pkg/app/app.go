package app

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/repository"
	"stream-recorder/internal/app/services/m3u8"
	"stream-recorder/internal/app/services/runner"
	"stream-recorder/internal/app/services/streamlink"
	"stream-recorder/internal/app/services/streams"
	"stream-recorder/pkg/db"
	embedded "stream-recorder/pkg/embed"
	"stream-recorder/pkg/logger"
)

type App struct {
	Router        *gin.Engine
	Cfg           *config.Config
	StreamersRepo *repository.StreamersRepository
	RunnerProcess *runner.Process
	Streamlink    *streamlink.Streamlink
	CheckStreams  *streams.Streams

	ActiveM3u8      map[string]*m3u8.M3u8
	ActiveStreamers map[string]bool
}

func New(workMode string) (*App, error) {
	logger.Init()

	if err := embedded.Init(); err != nil {
		return nil, err
	}

	if err := db.Init("db/stream-recorder.db"); err != nil {
		return nil, err
	}

	a := setupApplication(workMode)

	return a, nil
}

func setupApplication(workMode string) *App {
	var a = &App{}
	a.ActiveM3u8 = make(map[string]*m3u8.M3u8)
	a.ActiveStreamers = make(map[string]bool)

	var err error
	a.Cfg, err = config.New("config.json", workMode)
	if err != nil {
		logger.Fatal("Error loading config", zap.Error(err))
		return nil
	}

	logger.SetLogLevel(a.Cfg.LoggerLevel)

	//err = tmp.Clear("tmp")
	//if err != nil {
	//	logger.Error("Error clearing tmp", zap.Error(err))
	//	return nil
	//}

	a.StreamersRepo = repository.NewStreamers()
	a.RunnerProcess = runner.NewProcess()

	a.Streamlink = streamlink.New()
	a.CheckStreams = streams.New(a.StreamersRepo, a.Streamlink, a.RunnerProcess, a.Cfg, a.ActiveStreamers, a.ActiveM3u8)

	a.CheckStreams.Recovery()

	go a.CheckStreams.CheckingForStreams()
	return a
}
