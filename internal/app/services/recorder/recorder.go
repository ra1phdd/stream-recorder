package recorder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"stream-recorder/config"
	"stream-recorder/pkg/logger"
	"strings"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
	"go.uber.org/zap"
)

func Init(ctx context.Context, configEnv *config.ConfigurationEnv, username string, platform string, quality string) {
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			logger.Info("[" + username + "] Запись стрима прервана по запросу отмены")
			return
		}
	}()

	var m3u8 string
	switch platform {
	case "twitch":
		select {
		case <-ctx.Done():
			return
		default:
			m3u8 = FindLink("twitch.tv/"+username, username, quality, configEnv.TimeCheck)
		}
	default:
		logger.Fatal("["+username+"] Неизвестная платформа", zap.String("platform", platform))
	}

	path := GeneratePath(configEnv.RootPATH, username)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		logger.Fatal("[" + username + "] Ошибка создания папки для записи стрима")
	}

	logger.Info("[" + username + "] Начинаю запись стрима...")

	logFile, err := os.Create(fmt.Sprintf("logs/"+"ffmpeg-%s.log", username))
	if err != nil {
		fmt.Println("Ошибка создания файла ffmpeg.log", err)
		return
	}
	defer logFile.Close()

	var cmd *ffmpeg.Stream
	if configEnv.SplitSegments {
		go startSegmentMonitoring(done, username, configEnv.TimeSegment)

		cmd = ffmpeg.
			Input(m3u8).
			Output(path+"output_%03d.mov", ffmpeg.KwArgs{
				"c:v":              configEnv.VideoCodec,
				"c:a":              configEnv.AudioCodec,
				"segment_time":     configEnv.TimeSegment,
				"f":                "segment",
				"reset_timestamps": "1",
			}).
			WithErrorOutput(logFile)
	} else {
		cmd = ffmpeg.
			Input(m3u8).
			Output(path+"output_%03d.mov", ffmpeg.KwArgs{
				"c:v": configEnv.VideoCodec,
				"c:a": configEnv.AudioCodec,
			}).
			WithErrorOutput(logFile)
	}

	err = cmd.Run()

	if err != nil {
		logger.Error("["+username+"] Ошибка выполнения команды FFmpeg", zap.Error(err))
	} else {
		logger.Info("[" + username + "] Запись стрима окончена. Возвращаюсь к проверке наличия стрима...")
	}
	Init(ctx, configEnv, username, platform, quality)
}

func FindLink(link string, username string, quality string, timeCheck int) string {
	for {
		cmd := exec.Command("streamlink", "--stream-url", link, quality)

		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		resultChan := make(chan string)

		go func() {
			defer close(resultChan)

			time.Sleep(time.Duration(timeCheck) * time.Second)

			if strings.HasPrefix(stdoutBuf.String(), "https://") {
				resultChan <- stdoutBuf.String()
			} else if strings.Contains(stdoutBuf.String(), "No playable streams found on this URL") {
				logger.Info("[" + username + "] Стример не стримит, повторная проверка через " + fmt.Sprint(timeCheck) + " секунд")
				return
			} else if stdoutBuf.String() == "" {
				logger.Info("[" + username + "] Получение URL занимает больше времени, чем предполагалось... Ожидаю ответа")
				for {
					time.Sleep(500 * time.Millisecond)
					if stdoutBuf.String() != "" {
						resultChan <- stdoutBuf.String()
						break
					}
				}
			} else {
				logger.Error("["+stdoutBuf.String()+"] Неизвестная ошибка при получении URL потока", zap.Any("m3u8", stdoutBuf.String()))
				return
			}
		}()

		_ = cmd.Run()

		result := <-resultChan
		if result != "" {
			return result
		}
	}
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

func startSegmentMonitoring(done <-chan struct{}, username string, TimeSegment int) {
	time.Sleep(time.Duration(TimeSegment) * time.Second)
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
