package repository

import (
	"errors"
	"gorm.io/gorm"
	"log/slog"
	"stream-recorder/internal/app/models"
	"stream-recorder/pkg/logger"
)

type StreamersRepository struct {
	log *logger.Logger
	db  *gorm.DB
}

func NewStreamers(log *logger.Logger, db *gorm.DB) *StreamersRepository {
	return &StreamersRepository{
		log: log,
		db:  db,
	}
}

func (sr *StreamersRepository) Get() ([]models.Streamers, error) {
	sr.log.Trace("Entering Get method")

	var streamers []models.Streamers
	result := sr.db.Find(&streamers)

	if result.Error != nil {
		sr.log.Error("Failed to fetch streamers", result.Error)
		return nil, result.Error
	}
	if len(streamers) == 0 {
		sr.log.Warn("No streamers found in database")
		return nil, nil
	}

	sr.log.Debug("Streamers fetched successfully", slog.Any("streamers", streamers))
	return streamers, nil
}

func (sr *StreamersRepository) IsFoundStreamer(s models.Streamers) (bool, error) {
	sr.log.Trace("Entering IsFoundStreamer method", slog.Any("searchParams", s))

	var streamer models.Streamers
	result := sr.db.Where("username = ? AND platform = ?", s.Username, s.Platform).First(&streamer)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		sr.log.Debug("Streamer not found", slog.Any("streamer", s))
		return false, nil
	}
	if result.Error != nil {
		sr.log.Error("Database query failed", result.Error)
		return false, result.Error
	}

	sr.log.Debug("Streamer found successfully", slog.Int("id", streamer.ID), slog.Any("streamer", s))
	return true, nil
}

func (sr *StreamersRepository) Add(s models.Streamers) error {
	sr.log.Trace("Entering Add method", slog.Any("streamerToAdd", s))

	if err := sr.db.Create(&s).Error; err != nil {
		sr.log.Error("Failed to add new streamer", err, slog.Any("streamer", s))
		return err
	}

	sr.log.Debug("Streamer added successfully", slog.Any("streamer", s))
	return nil
}

func (sr *StreamersRepository) Delete(s models.Streamers) error {
	sr.log.Trace("Entering Delete method", slog.Any("streamerToDelete", s))

	result := sr.db.Where("platform = ? AND username = ?", s.Platform, s.Username).Delete(&models.Streamers{})
	if result.Error != nil {
		sr.log.Error("Failed to delete streamer", result.Error, slog.Any("streamer", s))
		return result.Error
	}

	if result.RowsAffected == 0 {
		sr.log.Warn("No rows deleted — streamer not found", slog.Any("streamer", s))
	} else {
		sr.log.Debug("Streamer deleted successfully", slog.Any("streamer", s))
	}
	return nil
}

func (sr *StreamersRepository) UpdateQuality(platform, username, quality string) error {
	sr.log.Trace("Entering UpdateQuality method", slog.String("platform", platform), slog.String("username", username), slog.String("quality", quality))

	result := sr.db.Model(&models.Streamers{}).
		Where("platform = ? AND username = ?", platform, username).
		Update("quality", quality)

	if result.Error != nil {
		sr.log.Error("Failed to update quality", result.Error, slog.String("platform", platform), slog.String("username", username))
		return result.Error
	}

	if result.RowsAffected == 0 {
		sr.log.Warn("No streamer found to update quality", slog.String("platform", platform), slog.String("username", username))
	} else {
		sr.log.Debug("Quality updated successfully", slog.String("platform", platform), slog.String("username", username))
	}
	return nil
}

func (sr *StreamersRepository) UpdateSegmentSettings(platform, username string, splitSegments bool, timeSegment int) error {
	sr.log.Trace("Entering UpdateSegmentSettings method",
		slog.String("platform", platform),
		slog.String("username", username),
		slog.Bool("split_segments", splitSegments),
		slog.Int("time_segment", timeSegment),
	)

	updateData := map[string]interface{}{
		"split_segments": splitSegments,
	}
	if timeSegment != 0 {
		updateData["time_segment"] = timeSegment
	}

	result := sr.db.Model(&models.Streamers{}).
		Where("platform = ? AND username = ?", platform, username).
		Updates(updateData)

	if result.Error != nil {
		sr.log.Error("Failed to update split_segments/time_segment", result.Error, slog.Any("updateData", updateData))
		return result.Error
	}

	if result.RowsAffected == 0 {
		sr.log.Warn("No streamer found to update split_segments/time_segment", slog.String("platform", platform), slog.String("username", username))
	} else {
		sr.log.Debug("split_segments/time_segment updated successfully", slog.String("platform", platform), slog.String("username", username))
	}
	return nil
}
