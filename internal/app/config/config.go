package config

import (
	"encoding/json"
	"fmt"
	"os"
	"stream-recorder/pkg/logger"
)

type Config struct {
	LoggerLevel            string `json:"logger_level"`
	TimeCheck              int    `json:"time_check"`
	FFmpegPATH             string `json:"ffmpeg_path"`
	MediaPATH              string `json:"media_path"`
	TempPATH               string `json:"temp_path"`
	AutoCleanMediaPATH     bool   `json:"auto_clean_media_path"`
	TimeAutoCleanMediaPATH int    `json:"time_auto_clean_media_path"`
	VideoCodec             string `json:"video_codec"`
	AudioCodec             string `json:"audio_codec"`
	FileFormat             string `json:"file_format"`

	// server
	Port    int    `json:"port"`
	GinMode string `json:"gin_mode"`
}

func New(configFile string, workMode string) (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := Config{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	cfg.setDefaults(workMode)
	cfg.NormalizeEnv(workMode)

	return &cfg, nil
}

func (c *Config) setDefaults(workMode string) {
	if c.LoggerLevel == "" {
		c.LoggerLevel = "warn"
	}
	if c.TimeCheck == 0 {
		c.TimeCheck = 15
	}
	if c.MediaPATH == "" {
		c.MediaPATH = "mnt"
	}
	if c.TempPATH == "" {
		c.TempPATH = "tmp"
	}
	if c.AutoCleanMediaPATH && c.TimeAutoCleanMediaPATH == 0 {
		c.TimeAutoCleanMediaPATH = 7
	}
	if c.VideoCodec == "" {
		c.VideoCodec = "copy"
	}
	if c.AudioCodec == "" {
		c.AudioCodec = "copy"
	}
	if c.FileFormat == "" {
		c.FileFormat = "mp4"
	}

	// server
	if workMode == "server" {
		if c.Port == 0 {
			c.Port = 8080
		}
		if c.GinMode == "" {
			c.GinMode = "release"
		}
	}
}

func (c *Config) NormalizeEnv(workMode string) {
	switch c.LoggerLevel {
	case "debug", "info", "warn", "error", "fatal":
	default:
		logger.Warn("Unknown LOGGER_LEVEL value. By default, 'info' is selected (available values - 'debug', 'info', 'warn', 'error', 'fatal')")
		c.LoggerLevel = "info"
	}

	if c.TimeCheck < 5 {
		logger.Warn("The time to check for a stream is too short. By default, 5 second is selected")
		c.TimeCheck = 5
	}

	if c.FFmpegPATH != "" {
		if _, err := os.Stat(c.FFmpegPATH); os.IsNotExist(err) {
			logger.Fatal("FFmpegPATH does not exist")
		}
	}

	if _, err := os.Stat(c.TempPATH); os.IsNotExist(err) {
		err = os.MkdirAll(c.TempPATH, 0755)
		if err != nil {
			logger.Fatal("Failed to create temp directory")
		}
	}

	if _, err := os.Stat(c.MediaPATH); os.IsNotExist(err) {
		err = os.MkdirAll(c.MediaPATH, 0755)
		if err != nil {
			logger.Fatal("Failed to create media directory")
		}
	}

	if c.TimeAutoCleanMediaPATH < 1 {
		logger.Warn("The time to auto clean media path is too short. By default, 1 day is selected")
		c.TimeAutoCleanMediaPATH = 1
	}

	if workMode == "server" {
		if c.Port < 0 || c.Port > 65535 {
			logger.Warn("The port must be between 1 and 65535, By default, 8080 is selected")
			c.Port = 8080
		}

		switch c.GinMode {
		case "debug", "release", "test":
		default:
			logger.Warn("Unknown GIN_MODE value. By default, 'release' is selected (available values - 'debug', 'release', 'test')")
			c.GinMode = "release"
		}
	}
}
