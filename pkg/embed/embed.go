package embed

import (
	"fmt"
	"log"
	"os"
	"stream-recorder/config"
)

var nameFile = make(map[string]string)

func Init(name string) error {
	name, err := getFileName(name)
	if err != nil {
		log.Fatalf("Error determining file name: %v", err)
		return err
	}

	tempFile, err := os.CreateTemp("", name)
	if err != nil {
		log.Fatalf("Ошибка создания временного файла: %v", err)
		return err
	}

	fileData, err := fs.ReadFile(fmt.Sprintf("bin/%s", name))
	if err != nil {
		log.Fatalf("Ошибка чтения бинарника: %v", err)
		return err
	}
	if _, err := tempFile.Write(fileData); err != nil {
		log.Fatalf("Ошибка записи бинарника Streamlink: %v", err)
		return err
	}
	tempFile.Close()

	// Установка прав на выполнение
	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		log.Fatalf("Ошибка установки прав на выполнение: %v", err)
		return err
	}

	nameFile[name] = tempFile.Name()

	return nil
}

func GetTempFileName(name string) string {
	var cfg config.Env
	var env string
	switch name {
	case "streamlink":
		env = cfg.StreamlinkPATH
	case "ffmpeg":
		env = cfg.FFmpegPATH
	}
	if env != "" {
		return env
	}

	name, err := getFileName(name)
	if err != nil {
		log.Fatalf("Error determining file name: %v", err)
	}

	return nameFile[name]
}
