package restStream

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"net/http"
	"strconv"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/services/m3u8"
	"stream-recorder/internal/app/services/models"
	"stream-recorder/internal/app/services/runner"
	"stream-recorder/pkg/logger"
	"strings"
	"time"
)

type Endpoint struct {
	am      map[string]*m3u8.M3u8
	as      map[string]bool
	limiter map[string]*rate.Limiter
	rp      *runner.Process
	cfg     *config.Config
}

func New(am map[string]*m3u8.M3u8, as map[string]bool, rp *runner.Process, cfg *config.Config) *Endpoint {
	return &Endpoint{
		am:      am,
		as:      as,
		limiter: make(map[string]*rate.Limiter),
		rp:      rp,
		cfg:     cfg,
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

	streamerID := fmt.Sprintf("%s-%s", s.Platform, s.Username)
	if _, exists := e.limiter[streamerID]; !exists {
		e.limiter[streamerID] = rate.NewLimiter(rate.Every(60*time.Second), 1)
	}

	if e.am[streamerID] == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "the streamer is not broadcasting live"})
		return
	}

	if e.limiter[streamerID].Allow() {
		e.am[streamerID].ChangeIsNeedCut(true)
		c.JSON(http.StatusOK, "success")
	} else {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "you can use cut no more than once per minute"})
		return
	}
}

func (e *Endpoint) DownloadM3u8Handler(c *gin.Context) {
	url := c.Query("url")
	isValid := (strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) && strings.HasSuffix(url, ".m3u8")
	if url == "" || !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is not a valid m3u8 link"})
		return
	}

	platform := c.Query("platform")
	username := c.Query("username")
	if platform == "" || username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform or username is empty"})
		return
	}

	splitSegments := false
	if splitSegmentsStr := c.Query("split_segments"); splitSegmentsStr != "" {
		parsed, err := strconv.ParseBool(splitSegmentsStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "split_segments contains an invalid value (expected true/false)"})
			return
		}
		splitSegments = parsed
	}

	timeSegment := 1800
	if timeSegmentStr := c.Query("time_segment"); timeSegmentStr != "" {
		parsed, err := strconv.Atoi(timeSegmentStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "time_segment contains an invalid value"})
			return
		}
		timeSegment = parsed
	}

	streamerID := fmt.Sprintf("%s-%s", platform, username)

	e.am[streamerID] = m3u8.New(platform, username, splitSegments, timeSegment, e.rp, e.cfg)
	err := e.am[streamerID].Run(url)
	if err != nil {
		logger.Error("Error running m3u8", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to run m3u8"})
	}
	e.as[streamerID] = false
}
