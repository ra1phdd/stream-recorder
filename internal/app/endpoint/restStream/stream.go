package restStream

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"net/http"
	"stream-recorder/internal/app/services/m3u8"
	"stream-recorder/internal/app/services/models"
	"time"
)

type Endpoint struct {
	am      map[string]*m3u8.M3u8
	limiter *rate.Limiter
}

func New(am map[string]*m3u8.M3u8) *Endpoint {
	return &Endpoint{
		am:      am,
		limiter: rate.NewLimiter(rate.Every(60*time.Second), 1),
	}
}

func (e *Endpoint) CutStreamHandler(c *gin.Context) {
	s := models.Streamers{
		Platform: c.Query("platform"),
		Username: c.Query("username"),
	}

	if s.Platform == "" || s.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform or username is empty"})
		return
	}

	if e.am[fmt.Sprintf("%s-%s", s.Platform, s.Username)] == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "the streamer is not broadcasting live"})
		return
	}

	if e.limiter.Allow() {
		e.am[fmt.Sprintf("%s-%s", s.Platform, s.Username)].ChangeIsNeedCut(true)
		c.JSON(http.StatusOK, "success")
	} else {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "you can use cut no more than once per minute"})
		return
	}
}
