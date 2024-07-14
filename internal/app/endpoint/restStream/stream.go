package restStream

import (
	"github.com/gin-gonic/gin"
	"stream-recorder/config"
	"stream-recorder/internal/app/services/tasks"
)

type Endpoint struct {
	Cfg *config.ConfigurationEnv
}

func (e Endpoint) CutStreamHandler(c *gin.Context) {
	tasks.CutTask(e.Cfg, c.Query("username"), c.Query("platform"), c.Query("quality"))
}
