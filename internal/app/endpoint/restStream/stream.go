package restStream

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

func (e Endpoint) CutStreamHandler(c *gin.Context) {
	logger.Debug("Получение запроса на разделение стрима")

	exists, user := config.GetUser(c.Query("username"))
	logger.Debug("Проверка на существование стримера в БД", zap.Bool("exists", exists))

	if exists {
		c.String(http.StatusOK, "успешно")
		tasks.CutTask(e.Cfg, user.Platform, user.Username, user.Quality)
	} else {
		c.String(http.StatusOK, "ошибка, стример отсутствует в БД")
	}
}
