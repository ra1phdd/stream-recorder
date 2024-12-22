package restStreamer

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"stream-recorder/internal/app/repository"
	"stream-recorder/internal/app/services/m3u8"
	"stream-recorder/internal/app/services/models"
)

type Endpoint struct {
	sr *repository.StreamersRepository
	am map[string]*m3u8.M3u8
}

func New(sr *repository.StreamersRepository, am map[string]*m3u8.M3u8) *Endpoint {
	return &Endpoint{
		sr: sr,
		am: am,
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
	var splitSegments bool
	var timeSegment int
	var err error
	if c.Query("split_segments") != "" {
		splitSegments, err = strconv.ParseBool(c.Query("split_segments"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "split_segments contains an invalid value (expected value is true/false)"})
			return
		}

		if c.Query("time_segment") != "" {
			timeSegment, err = strconv.Atoi(c.Query("time_segment"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "time_segment contains an invalid value"})
				return
			}
		} else {
			timeSegment = 1800
		}
	}

	s := models.Streamers{
		Platform:      c.Query("platform"),
		Username:      c.Query("username"),
		Quality:       c.Query("quality"),
		SplitSegments: splitSegments,
		TimeSegment:   timeSegment,
	}

	if s.Platform == "" || s.Username == "" || s.Quality == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform, username or quality is empty"})
		return
	}

	err = e.sr.Add(s)
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

	e.am[fmt.Sprintf("%s-%s", s.Platform, s.Username)].ChangeIsCancel(true)
	c.JSON(http.StatusOK, "success")
}
