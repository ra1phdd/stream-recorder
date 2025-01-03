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
	var streamers []models.Streamers
	platforms := strings.Split(c.Query("platform"), ",")
	usernames := strings.Split(c.Query("username"), ",")

	switch {
	case len(platforms) == len(usernames):
		for i := 0; i < len(platforms); i++ {
			streamers = append(streamers, models.Streamers{Platform: platforms[i], Username: usernames[i]})
		}
	case len(platforms) == 1 && len(usernames) > 1:
		for _, username := range usernames {
			streamers = append(streamers, models.Streamers{Platform: platforms[0], Username: username})
		}
	case len(usernames) == 1 && len(platforms) > 1:
		for _, platform := range platforms {
			streamers = append(streamers, models.Streamers{Platform: platform, Username: usernames[0]})
		}
	case len(platforms) == 1 && platforms[0] == "", len(usernames) == 1 && usernames[0] == "":
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform or username is empty"})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform and username counts do not match"})
		return
	}

	var success, failed []string
	for _, streamer := range streamers {
		streamerID := fmt.Sprintf("%s-%s", streamer.Platform, streamer.Username)

		if _, exists := e.limiter[streamerID]; !exists {
			e.limiter[streamerID] = rate.NewLimiter(rate.Every(60*time.Second), 1)
		}

		if e.am[streamerID] == nil {
			failed = append(failed, fmt.Sprintf("%s:%s (not live)", streamer.Platform, streamer.Username))
			continue
		}

		if e.limiter[streamerID].Allow() {
			e.am[streamerID].ChangeIsNeedCut(true)
			success = append(success, fmt.Sprintf("%s:%s", streamer.Platform, streamer.Username))
		} else {
			failed = append(failed, fmt.Sprintf("%s:%s (rate limit exceeded)", streamer.Platform, streamer.Username))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": strings.Join(success, ", "),
		"failed":  strings.Join(failed, ", "),
	})
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
