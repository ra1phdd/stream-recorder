package embed

import (
	"log"
	"stream-recorder/config"
)

func GetTempFileName(name string) string {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("%+v\n", err)
	}

	var env string
	switch name {
	case "streamlink":
		env = cfg.StreamlinkPATH
	case "ffmpeg":
		env = cfg.FFmpegPATH
	}

	return env
}
