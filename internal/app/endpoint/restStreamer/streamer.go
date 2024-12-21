package restStreamer

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"stream-recorder/internal/app/repository"
	"stream-recorder/internal/app/services/models"
)

type Endpoint struct {
	sr *repository.StreamersRepository
}

func New(sr *repository.StreamersRepository) *Endpoint {
	return &Endpoint{
		sr: sr,
	}
}

func (e *Endpoint) GetListStreamersHandler(c *gin.Context) {
	streamers, err := e.sr.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, streamers)
}

func (e *Endpoint) AddStreamerHandler(c *gin.Context) {
	s := models.Streamers{
		Platform: c.Query("platform"),
		Username: c.Query("username"),
		Quality:  c.Query("quality"),
	}

	if s.Platform == "" || s.Username == "" || s.Quality == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform, username or quality is empty"})
		return
	}

	err := e.sr.Add(s)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, "success")
}

func (e *Endpoint) DeleteStreamerHandler(c *gin.Context) {
	s := models.Streamers{
		Platform: c.Query("platform"),
		Username: c.Query("username"),
	}

	if s.Platform == "" || s.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform or username is empty"})
		return
	}

	err := e.sr.Delete(s)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, "success")
}
