package streamlink

import (
	"errors"
	"log/slog"
	"math/rand"
	"stream-recorder/internal/app/models"
	"stream-recorder/pkg/logger"
	"strings"
	"time"
)

type PlaylistProvider interface {
	GetMasterPlaylist(channel string) (string, error)
	FindMediaPlaylist(masterURL, quality string) (string, error)
	ParseM3u8(line string, m *models.StreamMetadata) (skipCount int, isSegment bool, segmentURL string)
}

type Streamlink struct {
	Platform PlaylistProvider
}

func New(log *logger.Logger, platformType string) *Streamlink {
	switch platformType {
	case "twitch":
		clientId := "kimne78kx3ncx6brgo4mv6wki5h1ko"
		deviceId, err := randomToken(32, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
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

func randomToken(length int, choices string) (string, error) {
	if length <= 0 {
		return "", errors.New("length must be greater than 0")
	}
	if len(choices) == 0 {
		return "", errors.New("choices string must not be empty")
	}

	var result strings.Builder
	choicesLen := len(choices)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < length; i++ {
		randomIndex := r.Intn(choicesLen)
		result.WriteByte(choices[randomIndex])
	}

	return result.String(), nil
}
