package config

import (
	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	"stream-recorder/pkg/logger"
	"strings"
)

type Config struct {
	Port          int    `env:"PORT" envDefault:"8080"`
	LoggerLevel   string `env:"LOGGER_LEVEL" envDefault:"warn"`
	GinMode       string `env:"GIN_MODE" envDefault:"release"`
	MediaPATH     string `env:"MEDIA_PATH,required"`
	TempPATH      string `env:"TMP_PATH" envDefault:"tmp"`
	FFmpegPATH    string `env:"FFMPEG_PATH"`
	SplitSegments bool   `env:"SPLIT_SEGMENTS" envDefault:"false"`
	TimeSegment   int    `env:"TIME_SEGMENT" envDefault:"1800"`
	TimeCheck     int    `env:"TIME_CHECK" envDefault:"15"`
	VideoCodec    string `env:"VIDEO_CODEC" envDefault:"copy"`
	AudioCodec    string `env:"AUDIO_CODEC" envDefault:"copy"`
	FileFormat    string `env:"FILE_FORMAT" envDefault:"mp4"`
}

func New(files ...string) (*Config, error) {
	err := godotenv.Load(files...)

	cfg := Config{}
	err = env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	cfg.NormalizeEnv()

	return &cfg, nil
}

func (c *Config) NormalizeEnv() {
	switch c.LoggerLevel {
	case "debug", "info", "warn", "error", "fatal":
		break
	default:
		logger.Warn("Unknown LOGGER_LEVEL value. By default, 'warn' is selected (available values - 'debug', 'info', 'warn', 'error', 'fatal')")
		c.LoggerLevel = "warn"
	}

	switch c.GinMode {
	case "debug", "release", "test":
		break
	default:
		logger.Warn("Unknown GIN_MODE value. By default, 'release' is selected (available values - 'debug', 'release', 'test')")
		c.GinMode = "release"
	}

	c.TempPATH = strings.TrimRight(c.TempPATH, "/\\?")
	c.MediaPATH = strings.TrimRight(c.MediaPATH, "/\\?")

	if c.TimeCheck < 5 {
		logger.Warn("The time to check for a stream is too short. By default, 5 seconds is selected")
		c.TimeCheck = 5
	}

	if c.TimeSegment < 60 {
		logger.Warn("The duration of one segment is too short. By default, 1 minute is selected")
		c.TimeSegment = 60
	}
}
