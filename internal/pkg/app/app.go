package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/handlers"
	"stream-recorder/internal/app/models"
	"stream-recorder/internal/app/repository"
	"stream-recorder/internal/app/services/scheduler"
	"stream-recorder/internal/app/services/state"
	"stream-recorder/internal/app/services/streamlink"
	"stream-recorder/internal/app/services/utils"
	"stream-recorder/pkg/logger"
	"time"
)

type App struct {
	log           *logger.Logger
	db            *gorm.DB
	cfg           *config.Config
	streamersRepo *repository.StreamersRepository
	streamlink    *streamlink.Streamlink
	scheduler     *scheduler.Scheduler
	state         *state.State
	utils         *utils.Utils
}

func New(workMode string) error {
	a := &App{
		log:   logger.New(),
		state: state.New(),
	}

	var err error
	a.db, err = gorm.Open(sqlite.Open("stream-recorder.db"), &gorm.Config{})
	if err != nil {
		return err
	}
	err = a.db.AutoMigrate(models.Streamers{})
	if err != nil {
		return err
	}

	a.cfg, err = config.New("config.json", a.log, workMode)
	if err != nil {
		a.log.Fatal("Error loading config", err)
		return nil
	}
	a.log.SetLogLevel(a.cfg.LoggerLevel)

	a.streamersRepo = repository.NewStreamers(a.log, a.db)
	a.streamlink = streamlink.New(a.log, "twitch")
	a.utils = utils.New(a.log)
	a.scheduler = scheduler.New(a.log, a.streamersRepo, a.streamlink, a.cfg, a.state, a.utils)

	go a.scheduler.Recovery()
	go a.scheduler.CheckingForStreams()

	go func() {
		for {
			err := a.utils.RemoveEmptyDirs(a.cfg.TempPATH)
			if err != nil {
				a.log.Error("Error clearing empty directory", err)
				return
			}
			time.Sleep(3 * time.Hour)
		}
	}()

	go func() {
		for {
			if !a.cfg.AutoCleanMediaPATH {
				break
			}
			err = a.utils.ClearToTime(a.cfg.MediaPATH, time.Duration(a.cfg.TimeAutoCleanMediaPATH)*24*time.Hour)
			if err != nil {
				a.log.Error("Error clearing directory to time", err)
				return
			}
			time.Sleep(3 * time.Hour)
		}
	}()

	if workMode == "server" {
		return setupServer(a)
	}

	return nil
}

func setupServer(a *App) error {
	gin.SetMode(a.cfg.GinMode)
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", time.Now().Add(-1*time.Second).Format(time.RFC1123))
		c.Next()
	})

	// регистрируем эндпоинты
	serviceStreamer := handlers.NewStreamer(a.log, a.streamersRepo, a.state)
	serviceStream := handlers.NewStream(a.log, a.state, a.cfg)

	// регистрируем маршруты
	r.GET("/streamer/list", serviceStreamer.GetStreamersHandler)
	r.GET("/streamer/add", serviceStreamer.AddStreamerHandler)
	r.GET("/streamer/update", serviceStreamer.UpdateStreamerHandler)
	r.GET("/streamer/delete", serviceStreamer.DeleteStreamerHandler)
	r.GET("/stream/cut", serviceStream.CutStreamHandler)
	//r.GET("/stream/download_m3u8", serviceStream.DownloadM3u8Handler)

	return runServer(r, a.cfg.Port)
}

func runServer(router *gin.Engine, port int) error {
	err := router.Run(fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	return nil
}
