package streamlink

import (
	"log/slog"
	"stream-recorder/internal/app/models"
	"stream-recorder/internal/app/services/utils"
	"stream-recorder/pkg/logger"
)

type PlaylistProvider interface {
	GetMasterPlaylist(channel string) (string, error)
	FindMediaPlaylist(masterURL, quality string) (string, error)
	ParseM3u8(line string, m *models.StreamMetadata) (skipCount int, isSegment bool, segmentURL string)
}

type Streamlink struct {
	Platform PlaylistProvider
}

func New(log *logger.Logger, u *utils.Utils, platformType string) *Streamlink {
	switch platformType {
	case "twitch":
		clientId := "kimne78kx3ncx6brgo4mv6wki5h1ko"
		deviceId, err := u.RandomToken(32, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
		if err != nil {
			log.Error("Failed generate random token", err)
			deviceId = "0cgX5cTZnLlpqmQjH71ndyWzrcAI6oal"
		}

		return &Streamlink{
			Platform: NewTwitch(log, clientId, deviceId),
		}
	default:
		log.Fatal("Unsupported platform type", nil, slog.String("platformType", platformType))
	}

	return nil
}
