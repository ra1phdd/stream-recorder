package repository

import (
	"database/sql"
	"errors"
	"go.uber.org/zap"
	"stream-recorder/internal/app/services/models"
	"stream-recorder/pkg/db"
	"stream-recorder/pkg/logger"
)

type StreamersRepository struct{}

func NewStreamers() *StreamersRepository {
	return &StreamersRepository{}
}

func (sr *StreamersRepository) Get() ([]models.Streamers, error) {
	logger.Debug("Fetching streamers")
	var s []models.Streamers

	rows, err := db.Conn.Query(`SELECT id, platform, username, quality, split_segments, time_segment FROM streamers`)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			logger.Error("Failed to close streamer rows", zap.Error(err))
		}
	}(rows)

	found := false
	var (
		id, timeSegment             int
		platform, username, quality string
		splitSegments               bool
	)
	for rows.Next() {
		err := rows.Scan(&id, &platform, &username, &quality, &splitSegments, &timeSegment)
		if err != nil {
			logger.Error("Failed to fetch streamers", zap.Error(err))
			return nil, err
		}

		item := models.Streamers{
			ID:            id,
			Platform:      platform,
			Username:      username,
			Quality:       quality,
			SplitSegments: splitSegments,
			TimeSegment:   timeSegment,
		}

		s = append(s, item)

		found = true
	}

	if !found {
		return nil, errors.New("no streamers found")
	}

	logger.Debug("Streamers fetched successfully", zap.Any("streamers", s))
	return s, nil
}

func (sr *StreamersRepository) GetById(id int) (models.Streamers, error) {
	logger.Debug("Fetching streamer by id", zap.Int("id", id))
	var s models.Streamers

	rows, err := db.Conn.Query(`SELECT platform, username, quality, split_segments, time_segment FROM streamers WHERE id = $1`, id)
	if err != nil {
		return models.Streamers{}, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			logger.Error("Failed to close streamer rows", zap.Error(err))
		}
	}(rows)

	err = rows.Scan(&s.Platform, &s.Username, &s.Quality, &s.SplitSegments, &s.TimeSegment)
	if err != nil {
		logger.Error("Failed to fetch streamers", zap.Error(err))
		return models.Streamers{}, err
	}

	logger.Debug("Streamer fetched successfully", zap.Any("streamer", s))
	return s, nil
}

func (sr *StreamersRepository) Add(s models.Streamers) error {
	logger.Debug("Adding new row in table streamers", zap.Any("streamers", s))

	_, err := db.Conn.Exec(`INSERT INTO streamers (platform, username, quality, split_segments, time_segment) VALUES ($1, $2, $3, $4, $5)`, s.Platform, s.Username, s.Quality, s.SplitSegments, s.TimeSegment)
	if err != nil {
		logger.Error("Failed to add new row in table streamers", zap.Error(err))
		return err
	}
	logger.Debug("New row in table streamers added successfully")

	return nil
}

func (sr *StreamersRepository) Delete(s models.Streamers) error {
	logger.Debug("Deleting row in table streamers", zap.Any("streamers", s))

	_, err := db.Conn.Exec(`DELETE FROM streamers WHERE platform = $1 AND username = $2`, s.Platform, s.Username)
	if err != nil {
		logger.Error("Failed to delete row in table streamers", zap.Error(err))
		return err
	}
	logger.Debug("Row in table streamers deleted successfully")

	return nil
}
