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

func (sr *StreamersRepository) IsFoundStreamer(s models.Streamers) (bool, error) {
	logger.Debug("Founding streamer in DB", zap.Any("s", s))

	var id int
	err := db.Conn.QueryRow(`SELECT id FROM streamers WHERE username = $1 AND platform = $2`, s.Username, s.Platform).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Debug("Streamer not found", zap.Any("streamer", s))
			return false, nil
		}
		logger.Error("Database query failed", zap.Error(err))
		return false, err
	}

	logger.Debug("Streamer found successfully", zap.Int("id", id), zap.Any("streamer", s))
	return true, nil
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

func (sr *StreamersRepository) UpdateQuality(platform, username, quality string) error {
	logger.Debug("Updating quality for streamer", zap.String("platform", platform), zap.String("username", username), zap.String("quality", quality))

	_, err := db.Conn.Exec(`UPDATE streamers SET quality = $1 WHERE platform = $2 AND username = $3`, quality, platform, username)
	if err != nil {
		logger.Error("Failed to update quality for streamer", zap.String("platform", platform), zap.String("username", username), zap.Error(err))
		return err
	}
	logger.Debug("Quality for streamer updated successfully", zap.String("platform", platform), zap.String("username", username))

	return nil
}

func (sr *StreamersRepository) UpdateSplitSegments(platform, username string, splitSegments bool, timeSegment int) error {
	logger.Debug("Updating split_segments and optionally time_segment for streamer",
		zap.String("platform", platform),
		zap.String("username", username),
		zap.Bool("split_segments", splitSegments),
		zap.Any("time_segment", timeSegment),
	)

	query := `UPDATE streamers SET split_segments = $1`
	args := []interface{}{splitSegments, platform, username}

	if timeSegment != 0 {
		query += `, time_segment = $2`
		args = append([]interface{}{splitSegments, timeSegment, platform, username}, args[3:]...)
	}

	query += ` WHERE platform = $3 AND username = $4`

	_, err := db.Conn.Exec(query, args...)
	if err != nil {
		logger.Error("Failed to update split_segments and/or time_segment for streamer",
			zap.String("platform", platform),
			zap.String("username", username),
			zap.Error(err),
		)
		return err
	}

	logger.Debug("Successfully updated split_segments and optionally time_segment for streamer",
		zap.String("platform", platform),
		zap.String("username", username),
	)

	return nil
}
