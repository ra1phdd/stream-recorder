package restStream

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"stream-recorder/config"
	"stream-recorder/internal/app/services/tasks"
)

type Endpoint struct {
	Cfg *config.ConfigurationEnv
}

func (e Endpoint) CutStreamHandler(c *gin.Context) {
	c.String(http.StatusOK, "success")
	tasks.CutTask(e.Cfg, c.Query("username"), c.Query("platform"), c.Query("quality"))
}
