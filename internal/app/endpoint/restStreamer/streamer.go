package restStreamer

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"stream-recorder/config"
	"stream-recorder/internal/app/services/tasks"
	"stream-recorder/pkg/logger"
)

type Endpoint struct {
	Cfg *config.Env
}

func (e Endpoint) GetListStreamersHandler(c *gin.Context) {
	logger.Debug("Получение запроса на вывод списка стримеров")
	cfg, err := config.ReadJSONConfig()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	} else {
		c.JSON(http.StatusOK, cfg)
	}
}

func (e Endpoint) AddStreamerHandler(c *gin.Context) {
	logger.Debug("Получение запроса на добавление стримера")
	user := config.StreamerConfig{
		Platform: c.Query("platform"),
		Username: c.Query("username"),
		Quality:  c.Query("quality"),
	}

	exists, _ := config.GetUser(user.Username)
	logger.Debug("Проверка на существование стримера в БД", zap.Bool("exists", exists))

	if !exists {
		err := config.AddUser(user)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		}

		tasks.StartTask(e.Cfg, user.Username, user.Platform, user.Quality)

		c.String(http.StatusOK, "успешно")
	} else {
		c.String(http.StatusOK, "ошибка, стример существует в БД")
	}
}

func (e Endpoint) DeleteStreamerHandler(c *gin.Context) {
	logger.Debug("Получение запроса на удаление стримера")
	user := config.StreamerConfig{
		Platform: c.Query("platform"),
		Username: c.Query("username"),
	}

	exists, _ := config.GetUser(user.Username)
	logger.Debug("Проверка на существование стримера в БД", zap.Bool("exists", exists))

	if exists {
		err := config.DeleteUser(user.Username)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		}

		tasks.StopTask(user.Username, user.Platform)

		c.String(http.StatusOK, "успешно")
	} else {
		c.String(http.StatusOK, "ошибка, стример отсутствует в БД")
	}
}
