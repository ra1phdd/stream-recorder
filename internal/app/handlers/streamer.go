package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"strconv"
	"stream-recorder/internal/app/models"
	"stream-recorder/internal/app/repository"
	"stream-recorder/internal/app/services/state"
	"stream-recorder/pkg/logger"
)

type StreamerHandler struct {
	log  *logger.Logger
	sr   *repository.StreamersRepository
	maps *state.State
}

func NewStreamer(log *logger.Logger, sr *repository.StreamersRepository, maps *state.State) *StreamerHandler {
	return &StreamerHandler{
		log:  log,
		sr:   sr,
		maps: maps,
	}
}

func (s *StreamerHandler) GetStreamersHandler(c *gin.Context) {
	s.log.Debug("Handling GetStreamers request")

	streamers, err := s.sr.Get()
	if err != nil {
		s.log.Error("Failed to retrieve streamers list", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.log.Info("Successfully fetched streamers list", slog.Int("count", len(streamers)))
	c.JSON(http.StatusOK, streamers)
}

func (s *StreamerHandler) AddStreamerHandler(c *gin.Context) {
	s.log.Debug("Handling AddStreamer request",
		slog.String("platform", c.Query("platform")),
		slog.String("username", c.Query("username")),
		slog.String("quality", c.Query("quality")),
		slog.String("split_segments", c.Query("split_segments")),
		slog.String("time_segment", c.Query("time_segment")),
	)

	var splitSegments bool
	var timeSegment int
	var err error
	if c.Query("split_segments") != "" {
		splitSegments, err = strconv.ParseBool(c.Query("split_segments"))
		if err != nil {
			s.log.Warn("Invalid split_segments value", err, slog.String("split_segments", c.Query("split_segments")))
			c.JSON(http.StatusBadRequest, gin.H{"error": "split_segments contains an invalid value (expected value is true/false)"})
			return
		}
		s.log.Debug("Parsed split_segments", slog.Bool("split_segments", splitSegments))

		timeSegment = 1800
		if c.Query("time_segment") != "" {
			timeSegment, err = strconv.Atoi(c.Query("time_segment"))
			if err != nil {
				s.log.Warn("Invalid time_segment value", err, slog.String("time_segment", c.Query("time_segment")))
				c.JSON(http.StatusBadRequest, gin.H{"error": "time_segment contains an invalid value"})
				return
			}
		}
		s.log.Debug("Parsed time_segment", slog.Int("time_segment", timeSegment))
	}

	st := models.Streamers{
		Platform:      c.Query("platform"),
		Username:      c.Query("username"),
		Quality:       c.Query("quality"),
		SplitSegments: splitSegments,
		TimeSegment:   timeSegment,
	}

	if st.Platform == "" || st.Username == "" || st.Quality == "" {
		s.log.Warn("Missing required query parameters", slog.String("platform", st.Platform), slog.String("username", st.Username), slog.String("quality", st.Quality))
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform, username or quality is empty"})
		return
	}

	s.log.Debug("Checking if streamer exists", slog.Any("streamer", st))
	isFound, err := s.sr.IsFoundStreamer(st)
	if err != nil {
		s.log.Error("Failed to check if streamer exists", err, slog.Any("streamer", st))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if isFound {
		s.log.Warn("Streamer already exists", slog.String("username", st.Username), slog.String("platform", st.Platform))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "the streamer already exists in the DB"})
		return
	}

	err = s.sr.Add(st)
	if err != nil {
		s.log.Error("Failed to add streamer", err, slog.Any("streamer", st))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.log.Info("Streamer added successfully", slog.String("username", st.Username), slog.String("platform", st.Platform))
	c.JSON(http.StatusOK, "success")
}

func (s *StreamerHandler) DeleteStreamerHandler(c *gin.Context) {
	s.log.Info("Handling DeleteStreamer request",
		slog.String("path", c.FullPath()),
		slog.String("method", c.Request.Method),
		slog.String("platform", c.Query("platform")),
		slog.String("username", c.Query("username")),
	)

	st := models.Streamers{
		Platform: c.Query("platform"),
		Username: c.Query("username"),
	}

	if st.Platform == "" || st.Username == "" {
		s.log.Warn("Missing required query parameters", slog.String("platform", st.Platform), slog.String("username", st.Username))
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform or username is empty"})
		return
	}

	s.log.Debug("Checking if streamer exists", slog.Any("streamer", st))
	isFound, err := s.sr.IsFoundStreamer(st)
	if err != nil {
		s.log.Error("Failed to check if streamer exists", err, slog.Any("streamer", st))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !isFound {
		s.log.Warn("Streamer does not exist exists", slog.String("username", st.Username), slog.String("platform", st.Platform))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "the streamer does not exist in the DB"})
		return
	}

	err = s.sr.Delete(st)
	if err != nil {
		s.log.Error("Failed to delete streamer", err, slog.Any("streamer", st))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	key := fmt.Sprintf("%s-%s", st.Platform, st.Username)
	s.maps.UpdateActiveStreamers(key, false)

	if s.maps.GetActiveM3u8(key) != nil {
		s.maps.GetActiveM3u8(key).ChangeIsCancel(true)
		s.log.Trace("Marked stream as cancelled", slog.String("key", key))
	}

	s.log.Info("Streamer deletion successful", slog.String("username", st.Username), slog.String("platform", st.Platform))
	c.JSON(http.StatusOK, "success")
}

func (s *StreamerHandler) UpdateStreamerHandler(c *gin.Context) {
	s.log.Info("Handling UpdateStreamer request",
		slog.String("path", c.FullPath()),
		slog.String("method", c.Request.Method),
		slog.String("platform", c.Query("platform")),
		slog.String("username", c.Query("username")),
	)

	platform := c.Query("platform")
	username := c.Query("username")
	if platform == "" || username == "" {
		s.log.Warn("Missing required query parameters", slog.String("platform", platform), slog.String("username", username))
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform or username is empty"})
		return
	}

	if quality := c.Query("quality"); quality != "" {
		s.log.Debug("Updating quality", slog.String("platform", platform), slog.String("username", username), slog.String("quality", quality))

		if err := s.sr.UpdateQuality(platform, username, quality); err != nil {
			s.log.Error("Failed to update quality", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		s.log.Info("Quality updated", slog.String("platform", platform), slog.String("username", username))
	}

	if splitSegmentsStr := c.Query("split_segments"); splitSegmentsStr != "" {
		splitSegments, err := strconv.ParseBool(splitSegmentsStr)
		if err != nil {
			s.log.Error("Invalid split_segments value", err, slog.String("value", splitSegmentsStr))
			c.JSON(http.StatusBadRequest, gin.H{"error": "split_segments contains an invalid value (expected true/false)"})
			return
		}

		var timeSegment int
		if timeSegmentStr := c.Query("time_segment"); timeSegmentStr != "" {
			timeSegment, err = strconv.Atoi(timeSegmentStr)
			if err != nil {
				s.log.Error("Invalid time_segment value", err, slog.String("value", timeSegmentStr))
				c.JSON(http.StatusBadRequest, gin.H{"error": "time_segment contains an invalid value"})
				return
			}
			s.log.Debug("Parsed time_segment", slog.Int("time_segment", timeSegment))
		}

		s.log.Debug("Updating segment settings",
			slog.String("platform", platform),
			slog.String("username", username),
			slog.Bool("split_segments", splitSegments),
			slog.Int("time_segment", timeSegment),
		)
		if err := s.sr.UpdateSegmentSettings(platform, username, splitSegments, timeSegment); err != nil {
			s.log.Error("Failed to update segment settings", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s.log.Debug("Segment settings updated", slog.String("platform", platform), slog.String("username", username))
	}

	s.log.Info("Streamer update successful", slog.String("platform", platform), slog.String("username", username))
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
