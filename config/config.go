package config

import (
	"encoding/json"
	"fmt"
	"os"
	"stream-recorder/pkg/logger"
	"sync"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

var mu sync.RWMutex
var config *ConfigurationJSON

type ConfigurationEnv struct {
	Port          string `env:"PORT" envDefault:"8080"`
	LoggerLevel   string `env:"LOGGER_LEVEL" envDefault:"warn"`
	GinMode       string `env:"GIN_MODE" envDefault:"release"`
	RootPATH      string `env:"ROOT_PATH,required"`
	SplitSegments bool   `env:"SPLIT_SEGMENTS" envDefault:"false"`
	TimeSegment   int    `env:"TIME_SEGMENT" envDefault:"1800"`
	TimeCheck     int    `env:"TIME_CHECK" envDefault:"15"`
	VideoCodec    string `env:"VIDEO_CODEC" envDefault:"copy"`
	AudioCodec    string `env:"AUDIO_CODEC" envDefault:"copy"`
	FileFormat    string `env:"FILE_FORMAT" envDefault:"mp4"`
}

type UserConfiguration struct {
	Platform string `json:"platform"`
	Username string `json:"username"`
	Quality  string `json:"quality"`
}

type ConfigurationJSON struct {
	Users []UserConfiguration `json:"users"`
}

func (c *ConfigurationEnv) NormalizeEnv() {
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

func NewConfig(files ...string) (*ConfigurationEnv, error) {
	err := godotenv.Load(files...)

	cfg := ConfigurationEnv{}
	err = env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	cfg.NormalizeEnv()

	return &cfg, nil
}

func ReadJSONConfig() (*ConfigurationJSON, error) {
	var cfg ConfigurationJSON
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

// Функция для обновления конфигурации
func UpdateJSONConfig() error {
	cfg, err := ReadJSONConfig()
	if err != nil {
		return err
	}

	mu.Lock()
	config = cfg
	mu.Unlock()

	fmt.Println("Configuration reloaded:", config)
	return nil
}

// Запись конфигурации в файл
func WriteConfig(cfg *ConfigurationJSON) error {
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

func AddUser(user UserConfiguration) error {
	mu.Lock()
	defer mu.Unlock()

	config.Users = append(config.Users, user)

	return WriteConfig(config)
}

func DeleteUser(username string) error {
	mu.Lock()
	defer mu.Unlock()

	for i, user := range config.Users {
		if user.Username == username {
			config.Users = append(config.Users[:i], config.Users[i+1:]...)
			return WriteConfig(config)
		}
	}
	return fmt.Errorf("User not found")
}

func GetUser(username string) error {
	mu.Lock()
	defer mu.Unlock()

	for i, user := range config.Users {
		if user.Username == username {
			config.Users = append(config.Users[:i], config.Users[i+1:]...)
			return WriteConfig(config)
		}
	}
	return fmt.Errorf("User not found")
}
