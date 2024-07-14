package restStreamer

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"stream-recorder/config"
	"stream-recorder/internal/app/services/tasks"
)

type Endpoint struct {
	Cfg *config.ConfigurationEnv
}

func (e Endpoint) GetListStreamersHandler(c *gin.Context) {
	cfg, err := config.ReadJSONConfig()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.JSON(http.StatusOK, cfg)
}

func (e Endpoint) AddStreamerHandler(c *gin.Context) {
	user := config.UserConfiguration{
		Platform: c.Query("platform"),
		Username: c.Query("username"),
		Quality:  c.Query("quality"),
	}

	err := config.AddUser(user)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	tasks.StartTask(e.Cfg, user.Username, user.Platform, user.Quality)

	c.String(http.StatusOK, "success")
}

func (e Endpoint) DeleteStreamerHandler(c *gin.Context) {
	err := config.DeleteUser(c.Query("username"))
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	tasks.StopTask(c.Query("username"), c.Query("platform"))

	c.String(http.StatusOK, "success")
}
