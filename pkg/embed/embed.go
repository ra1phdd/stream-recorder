package embed

import (
	"embed"
	"fmt"
	"log"
	"os"
	"stream-recorder/internal/app/config"
)

//go:embed db/stream-recorder.db
var embeddedDB []byte

var nameFile = make(map[string]string)

func Init() error {
	DB()

	err := CreateFile(getFileFfmpeg(), "ffmpeg", fsFfmpeg)
	if err != nil {
		return err
	}

	return nil
}

func CreateFile(name string, dir string, fs embed.FS) error {
	tempFile, err := os.CreateTemp("", name)
	if err != nil {
		log.Fatalf("Ошибка создания временного файла: %v", err)
		return err
	}

	fileData, err := fs.ReadFile(fmt.Sprintf("%s/%s", dir, name))
	if err != nil {
		log.Fatalf("Ошибка чтения бинарника: %v", err)
		return err
	}
	if _, err := tempFile.Write(fileData); err != nil {
		log.Fatalf("Ошибка записи бинарника Streamlink: %v", err)
		return err
	}
	tempFile.Close()

	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		log.Fatalf("Ошибка установки прав на выполнение: %v", err)
		return err
	}

	nameFile[name] = tempFile.Name()

	return nil
}

func GetTempFileName(name string) string {
	c, err := config.New()
	if err == nil && c.FFmpegPATH != "" {
		return c.FFmpegPATH
	}

	switch name {
	case "ffmpeg":
		return nameFile[getFileFfmpeg()]
	default:
		return ""
	}
}

func DB() {
	dbDir := "db"
	FileDB := dbDir + "/stream-recorder.db"

	if err := os.MkdirAll(dbDir, 0755); err != nil {
		fmt.Printf("Ошибка создания директории: %v\n", err)
		return
	}

	if _, err := os.Stat(FileDB); os.IsNotExist(err) {
		err = os.WriteFile(FileDB, embeddedDB, 0644)
		if err != nil {
			fmt.Printf("Ошибка записи файла: %v\n", err)
			return
		}
	}
}
