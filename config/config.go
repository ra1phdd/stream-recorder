package config

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"os"
	"stream-recorder/pkg/logger"
	"sync"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

var mu sync.RWMutex
var config *JSON

type Env struct {
	Port           string `env:"PORT" envDefault:"8080"`
	LoggerLevel    string `env:"LOGGER_LEVEL" envDefault:"warn"`
	GinMode        string `env:"GIN_MODE" envDefault:"release"`
	RootPATH       string `env:"ROOT_PATH,required"`
	StreamlinkPATH string `env:"STREAMLINK_PATH"`
	FFmpegPATH     string `env:"FFMPEG_PATH"`
	SplitSegments  bool   `env:"SPLIT_SEGMENTS" envDefault:"false"`
	TimeSegment    int    `env:"TIME_SEGMENT" envDefault:"1800"`
	TimeCheck      int    `env:"TIME_CHECK" envDefault:"15"`
	VideoCodec     string `env:"VIDEO_CODEC" envDefault:"copy"`
	AudioCodec     string `env:"AUDIO_CODEC" envDefault:"copy"`
	FileFormat     string `env:"FILE_FORMAT" envDefault:"mp4"`
}

type StreamerConfig struct {
	Platform string `json:"platform"`
	Username string `json:"username"`
	Quality  string `json:"quality"`
}

type JSON struct {
	Streamers []StreamerConfig `json:"streamers"`
}

func (c *Env) NormalizeEnv() {
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

func NewConfig(files ...string) (*Env, error) {
	err := godotenv.Load(files...)

	cfg := Env{}
	err = env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	cfg.NormalizeEnv()

	return &cfg, nil
}

func ReadJSONConfig() (*JSON, error) {
	var cfg JSON
	file, err := os.Open("streamers.json")
	if err != nil {
		return &cfg, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		return &cfg, err
	}

	return &cfg, nil
}

func UpdateJSONConfig() error {
	cfg, err := ReadJSONConfig()
	if err != nil {
		return err
	}

	mu.Lock()
	config = cfg
	mu.Unlock()

	logger.Debug("Обновление JSON-конфигурации", zap.Any("cfg", cfg))
	return nil
}

func WriteConfig(cfg *JSON) error {
	file, err := os.Create("streamers.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(cfg)
	if err != nil {
		return err
	}

	return nil
}

func AddUser(user StreamerConfig) error {
	mu.Lock()
	defer mu.Unlock()

	config.Streamers = append(config.Streamers, user)

	logger.Debug("Добавление стримера в JSON", zap.Any("streamer", user))
	return WriteConfig(config)
}

func DeleteUser(username string) error {
	mu.Lock()
	defer mu.Unlock()

	for i, user := range config.Streamers {
		if user.Username == username {
			config.Streamers = append(config.Streamers[:i], config.Streamers[i+1:]...)
			logger.Debug("Удаление стримера из JSON", zap.Any("streamer", user))
			return WriteConfig(config)
		}
	}
	return fmt.Errorf("стример не найден в JSON")
}

func GetUser(username string) (bool, StreamerConfig) {
	mu.Lock()
	defer mu.Unlock()

	for _, user := range config.Streamers {
		if user.Username == username {
			return true, StreamerConfig{
				Platform: user.Platform,
				Username: user.Username,
				Quality:  user.Quality,
			}
		}
	}
	return false, StreamerConfig{}
}
