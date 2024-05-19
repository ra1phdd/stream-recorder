package recorder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"stream-recorder/internal/config"
	"stream-recorder/pkg/logger"
	"strings"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
	"go.uber.org/zap"
)

func Init(configEnv *config.Configuration, configJSON *config.Configuration) {
	var m3u8 string
	switch configJSON.Platform {
	case "twitch":
		for {
			m3u8 = FindLink("twitch.tv/" + configJSON.Username)
			if strings.HasPrefix(m3u8, "https://") {
				break
			} else if strings.Contains(m3u8, "No playable streams found on this URL") {
				logger.Info("[" + configJSON.Username + "] Стример не стримит, повторная проверка через " + fmt.Sprint(configEnv.TimeCheck) + " секунд")
			} else {
				logger.Error("["+configJSON.Username+"] Неизвестная ошибка при получении URL потока", zap.Any("m3u8", m3u8))
			}
			time.Sleep(time.Duration(configEnv.TimeCheck) * time.Second)
		}
	default:
		logger.Fatal("["+configJSON.Username+"] Неизвестная платформа", zap.String("platform", configJSON.Platform))
	}

	path := GeneratePath(configJSON.RootPATH, configJSON.Username)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		logger.Fatal("[" + configJSON.Username + "] Ошибка создания папки для записи стрима")
	}

	var stderr, stdout bytes.Buffer

	// Канал завершения
	done := make(chan bool)

	logger.Info("[" + configJSON.Username + "] Начинаю запись стрима...")

	var cmd *ffmpeg.Stream
	if configEnv.SplitSegments {
		// Запускаем мониторинг в отдельной горутине
		go startSegmentMonitoring(done, configJSON.Username, configEnv.TimeSegment)

		cmd = ffmpeg.
			Input(m3u8).
			Output(path+"output_%03d.mov", ffmpeg.KwArgs{
				"c:v":              configEnv.VideoCodec,
				"c:a":              configEnv.AudioCodec,
				"segment_time":     configEnv.TimeSegment,
				"f":                "segment",
				"reset_timestamps": "1",
			}).
			WithErrorOutput(&stderr).
			WithOutput(&stdout)
	} else {
		cmd = ffmpeg.
			Input(m3u8).
			Output(path+"output_%03d.mov", ffmpeg.KwArgs{
				"c:v": configEnv.VideoCodec,
				"c:a": configEnv.AudioCodec,
			}).
			WithErrorOutput(&stderr).
			WithOutput(&stdout)
	}

	err = cmd.Run()

	// Отправляем сигнал завершения горутине
	close(done)

	// Обрабатываем вывод прогресса
	if err != nil {
		logger.Error("["+configJSON.Username+"] Ошибка выполнения команды FFmpeg", zap.Error(err), zap.Any("Вывод ffmpeg", stderr.String()))
	} else {
		logger.Info("[" + configJSON.Username + "] Запись стрима окончена. Возвращаюсь к проверке наличия стрима...")
	}
}

func FindLink(link string) string {
	// Добавляем контекст с тайм-аутом
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "streamlink", "--stream-url", link, "best")

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	_ = cmd.Run()

	time.Sleep(1 * time.Second)

	return stdoutBuf.String()
}

func GeneratePath(rootPath string, username string) string {
	currentTime := time.Now()
	formattedTime := currentTime.Format("2006-01-02_15-04-05")

	var path string
	switch runtime.GOOS {
	case "windows":
		path = rootPath + "\\" + "twitch_" + username + "_" + formattedTime + "\\"
	case "linux", "darwin":
		path = rootPath + "/" + "twitch_" + username + "_" + formattedTime + "/"
	default:
		logger.Fatal("["+username+"] Неизвестная система", zap.Any("ОС", runtime.GOOS))
	}

	return path
}

func startSegmentMonitoring(done <-chan bool, username string, TimeSegment int) {
	time.Sleep(15 * time.Second)
	for {
		select {
		case <-done:
			logger.Info("[" + username + "] Запись стрима завершена")
			return
		case <-time.After(time.Duration(TimeSegment) * time.Second):
			logger.Info("[" + username + "] Сегмент записан, перехожу к следующему")
		}
	}
}
