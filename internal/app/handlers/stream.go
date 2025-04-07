package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"log/slog"
	"net/http"
	"stream-recorder/internal/app/config"
	"stream-recorder/internal/app/models"
	"stream-recorder/internal/app/services/state"
	"stream-recorder/pkg/logger"
	"strings"
	"time"
)

type StreamHandler struct {
	log  *logger.Logger
	maps *state.State
	cfg  *config.Config

	limiter map[string]*rate.Limiter
}

func NewStream(log *logger.Logger, maps *state.State, cfg *config.Config) *StreamHandler {
	return &StreamHandler{
		log:     log,
		maps:    maps,
		cfg:     cfg,
		limiter: make(map[string]*rate.Limiter),
	}
}

func (s *StreamHandler) CutStreamHandler(c *gin.Context) {
	platforms := strings.Split(c.Query("platform"), ",")
	usernames := strings.Split(c.Query("username"), ",")

	s.log.Debug("Received cut stream request", slog.Any("platforms", platforms), slog.Any("usernames", usernames))
	var streamers []models.Streamers
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
		s.log.Warn("Empty platform and username in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform or username is empty"})
		return
	default:
		s.log.Warn("Mismatched platform and username counts", slog.Int("platformCount", len(platforms)), slog.Int("userCount", len(usernames)))
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform and username counts do not match"})
		return
	}

	var success, failed []string
	for _, streamer := range streamers {
		key := fmt.Sprintf("%s-%s", streamer.Platform, streamer.Username)
		s.log.Trace(fmt.Sprintf("[%s/%s] Processing streamer", streamer.Platform, streamer.Username))

		if _, exists := s.limiter[key]; !exists {
			s.log.Debug(fmt.Sprintf("[%s/%s] Initializing rate limiter", streamer.Platform, streamer.Username))
			s.limiter[key] = rate.NewLimiter(rate.Every(60*time.Second), 1)
		}

		if s.maps.GetActiveM3u8(key) == nil {
			s.log.Info(fmt.Sprintf("[%s/%s] Streamer is not live", streamer.Platform, streamer.Username))
			failed = append(failed, fmt.Sprintf("%s:%s (not live)", streamer.Platform, streamer.Username))
			continue
		}

		if s.limiter[key].Allow() {
			s.maps.GetActiveM3u8(key).ChangeIsNeedCut(true)
			s.log.Info(fmt.Sprintf("[%s/%s] Stream marked for cut", streamer.Platform, streamer.Username))
			success = append(success, fmt.Sprintf("%s:%s", streamer.Platform, streamer.Username))
		} else {
			s.log.Warn(fmt.Sprintf("[%s/%s] Rate limit exceeded", streamer.Platform, streamer.Username))
			failed = append(failed, fmt.Sprintf("%s:%s (rate limit exceeded)", streamer.Platform, streamer.Username))
		}
	}

	s.log.Debug("Cut stream result", slog.Any("success", success), slog.Any("failed", failed))
	c.JSON(http.StatusOK, gin.H{
		"success": strings.Join(success, ", "),
		"failed":  strings.Join(failed, ", "),
	})
}

//func (s *StreamHandler) DownloadM3u8Handler(c *gin.Context) {
//	url := c.Query("url")
//	isValid := (strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) && strings.HasSuffix(url, ".m3u8")
//	if url == "" || !isValid {
//		s.log.Warn("Invalid m3u8 URL requested", slog.String("url", url), slog.Bool("is_valid", isValid))
//		c.JSON(http.StatusBadRequest, gin.H{"error": "url is not a valid m3u8 link"})
//		return
//	}
//	s.log.Debug("Received valid m3u8 URL", slog.String("url", url))
//
//	platform := c.Query("platform")
//	username := c.Query("username")
//	if platform == "" || username == "" {
//		s.log.Warn("Empty platform or username", slog.String("platform", platform), slog.String("username", username))
//		c.JSON(http.StatusBadRequest, gin.H{"error": "platform or username is empty"})
//		return
//	}
//
//
//	splitSegments := false
//	if splitSegmentsStr := c.Query("split_segments"); splitSegmentsStr != "" {
//		parsed, err := strconv.ParseBool(splitSegmentsStr)
//		if err != nil {
//			c.JSON(http.StatusBadRequest, gin.H{"error": "split_segments contains an invalid value (expected true/false)"})
//			return
//		}
//		splitSegments = parsed
//	}
//
//	timeSegment := 1800
//	if timeSegmentStr := c.Query("time_segment"); timeSegmentStr != "" {
//		parsed, err := strconv.Atoi(timeSegmentStr)
//		if err != nil {
//			c.JSON(http.StatusBadRequest, gin.H{"error": "time_segment contains an invalid value"})
//			return
//		}
//		timeSegment = parsed
//	}
//
//	key := fmt.Sprintf("%s-%s", platform, username)
//	val, err := m3u8.New(s.log, platform, username, splitSegments, timeSegment, s.cfg)
//	if err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//		return
//	}
//	s.maps.UpdateActiveM3u8(key, val)
//
//	err = s.maps.GetActiveM3u8(key).Run(url)
//	if err != nil {
//		s.log.Error("Error running m3u8", err)
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to run m3u8"})
//	}
//	s.maps.UpdateActiveStreamers(key, false)
//}
