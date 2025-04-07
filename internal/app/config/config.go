package config

import (
	"encoding/json"
	"log/slog"
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
	BufferSize             int    `json:"buffer_size"`
	VideoCodec             string `json:"video_codec"`
	AudioCodec             string `json:"audio_codec"`
	FileFormat             string `json:"file_format"`

	// server
	Port    int    `json:"port"`
	GinMode string `json:"gin_mode"`
}

func New(configFile string, log *logger.Logger, workMode string) (*Config, error) {
	log.Debug("Initializing config from file", slog.Any("file", configFile), slog.String("mode", workMode))

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Error("Failed to read config file", err, slog.Any("file", configFile))
		return nil, err
	}
	log.Debug("Successfully read config file", slog.Any("file", configFile))

	cfg := Config{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Error("Failed to parse config JSON from file", err, slog.Any("file", configFile))
		return nil, err
	}
	log.Debug("Successfully parsed config JSON")

	log.Trace("Setting default config values")
	cfg.setDefaults(workMode)
	log.Debug("Defaults have been set")

	log.Trace("Normalizing environment settings")
	cfg.normalizeEnv(log, workMode)
	log.Debug("Environment has been normalized")

	log.Info("Config initialized successfully", slog.Any("file", configFile), slog.String("mode", workMode))
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
	if c.BufferSize == 0 {
		c.BufferSize = 32
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

func (c *Config) normalizeEnv(log *logger.Logger, workMode string) {
	switch c.LoggerLevel {
	case "debug", "info", "warn", "error", "fatal":
		break
	default:
		log.Warn("Unknown LOGGER_LEVEL value. By default, 'info' is selected (available values - 'debug', 'info', 'warn', 'error', 'fatal')")
		c.LoggerLevel = "info"
	}

	if c.TimeCheck < 5 {
		log.Warn("The time to check for a stream is too short. By default, 5 second is selected")
		c.TimeCheck = 5
	}

	if c.FFmpegPATH != "" {
		if _, err := os.Stat(c.FFmpegPATH); os.IsNotExist(err) {
			log.Fatal("FFmpegPATH does not exist", err)
		}
	}

	if _, err := os.Stat(c.TempPATH); os.IsNotExist(err) {
		err = os.MkdirAll(c.TempPATH, 0755)
		if err != nil {
			log.Fatal("Failed to create temp directory", err)
		}
	}

	if _, err := os.Stat(c.MediaPATH); os.IsNotExist(err) {
		err = os.MkdirAll(c.MediaPATH, 0755)
		if err != nil {
			log.Fatal("Failed to create media directory", err)
		}
	}

	if c.TimeAutoCleanMediaPATH < 1 {
		log.Warn("The time to auto clean media path is too short. By default, 1 day is selected")
		c.TimeAutoCleanMediaPATH = 1
	}

	if c.BufferSize < 32 {
		log.Warn("The buffer size cannot be less than 32 megabytes. 32 megabytes is selected by default.")
		c.BufferSize = 32
	}

	if workMode == "server" {
		if c.Port < 0 || c.Port > 65535 {
			log.Warn("The port must be between 1 and 65535, By default, 8080 is selected")
			c.Port = 8080
		}

		switch c.GinMode {
		case "debug", "release", "test":
			break
		default:
			log.Warn("Unknown GIN_MODE value. By default, 'release' is selected (available values - 'debug', 'release', 'test')")
			c.GinMode = "release"
		}
	}
}
