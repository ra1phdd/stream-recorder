package config

import (
	"encoding/json"
	"os"
	"stream-recorder/pkg/logger"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

type Configuration struct {
	RootPATH      string `json:"root_path"`
	Platform      string `json:"platform"`
	Username      string `json:"username"`
	Quality       string `json:"quality"`
	SplitSegments bool   `env:"SPLIT_SEGMENTS" envDefault:"false"`
	TimeSegment   int    `env:"TIME_SEGMENT" envDefault:"1800"`
	TimeCheck     int    `env:"TIME_CHECK" envDefault:"15"`
	VideoCodec    string `env:"VIDEO_CODEC" envDefault:"copy"`
	AudioCodec    string `env:"AUDIO_CODEC" envDefault:"copy"`
	FileFormat    string `env:"FILE_FORMAT" envDefault:"mp4"`
}

func (c *Configuration) NormalizeJSON() {
	switch c.Platform {
	case "twitch", "kick", "youtube", "vkplay":
		break
	default:
		logger.Warn("Неизвестная платформа. По умолчанию выбран Twitch")
		c.Platform = "twitch"
	}
	switch c.Quality {
	case "1080p", "720p", "480p", "360p", "160p", "best", "worst":
		break
	default:
		logger.Warn("Неизвестное качество. По умолчанию выбрано самое лучшее")
		c.Quality = "best"
	}
}

func (c *Configuration) NormalizeEnv() {
	switch c.SplitSegments {
	case true, false:
		break
	default:
		logger.Warn("Неизвестное значение SplitSegments. По умолчанию выбрано false")
		c.SplitSegments = false
	}
	if c.TimeCheck < 5 {
		logger.Warn("Слишком малое время проверки наличия стрима. По умолчанию выбрано 5 секунд")
		c.TimeCheck = 5
	}
	if c.TimeSegment < 300 {
		logger.Warn("Слишком малая длительность одного сегмента. По умолчанию выбрано 5 минут")
		c.TimeSegment = 300
	}
}

func NewJsonConfig(configPath string) (*Configuration, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Configuration
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	cfg.NormalizeJSON()

	return &cfg, nil
}

func NewEnvConfig(files ...string) (*Configuration, error) {
	err := godotenv.Load(files...)

	cfg := Configuration{}
	err = env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	cfg.NormalizeEnv()

	return &cfg, nil
}
