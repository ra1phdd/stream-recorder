package recorder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"stream-recorder/config"
	"stream-recorder/pkg/logger"
	"strings"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
	"go.uber.org/zap"
)

func Init(ctx context.Context, configEnv *config.Env, platform string, username string, quality string) {
	logger.Debug("Инициализация горутины проверки наличия и записи стрима", zap.String("platform", platform), zap.String("username", username), zap.String("quality", quality))
	done := make(chan struct{})
	defer close(done)

	go watchContext(ctx, username, done)

	var m3u8 string
	for {
		select {
		case <-ctx.Done():
			return
		default:
			m3u8 = findLink(platform, username, quality, configEnv.TimeCheck)
		}

		path := generatePath(configEnv.RootPATH, platform, username)
		err := os.MkdirAll(path, 0755)
		if err != nil {
			logger.Fatal("[" + username + "] Ошибка создания папки для записи стрима")
		}

		logger.Info("[" + username + "] Начинаю запись стрима...")

		err = runFFmpegCommand(m3u8, path, username, configEnv, done)
		if err != nil {
			logger.Error("["+username+"] Ошибка выполнения команды FFmpeg", zap.Error(err))
		} else {
			logger.Info("[" + username + "] Запись стрима окончена. Возвращаюсь к проверке наличия стрима...")
		}
	}
}

func findLink(platform string, username string, quality string, timeCheck int) string {
	logger.Debug("Поиск ссылки потока", zap.String("platform", platform), zap.String("username", username), zap.Int("timeCheck", timeCheck))
	var link string
	switch platform {
	case "twitch":
		link = "twitch.tv/" + username
	case "youtube":
		link = "youtube.com/@" + username + "/live"
	case "kick":
		link = "kick.com/" + username
	default:
		logger.Fatal("["+username+"] Неизвестная платформа", zap.String("platform", platform))
	}
	logger.Debug("Полученная ссылка стрима", zap.String("link", link))

	for {
		cmd := exec.Command("streamlink", "--stream-url", link, quality)

		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		err := cmd.Start()
		if err != nil {
			logger.Error("Ошибка инициализация команды streamlink", zap.String("link", link), zap.String("quality", quality), zap.Error(err))
			time.Sleep(time.Duration(timeCheck) * time.Second)
			continue
		}

		done := make(chan error)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-time.After(time.Duration(timeCheck) * time.Second):
			logger.Info("[" + username + "] Получение URL занимает больше времени, чем предполагалось... Ожидаю ответа")
		case <-done:
			stdout := stdoutBuf.String()

			if strings.HasPrefix(stdout, "https://") {
				return stdout
			} else if strings.Contains(stdout, "No playable streams found on this URL") {
				logger.Info("[" + username + "] Стример не стримит, повторная проверка через " + fmt.Sprint(timeCheck) + " секунд")
				time.Sleep(time.Duration(timeCheck) * time.Second)
				continue
			} else {
				logger.Error("["+username+"] Неизвестная ошибка при получении URL потока", zap.Any("stdout", stdout))
				time.Sleep(time.Duration(timeCheck) * time.Second)
				continue
			}
		}
	}
}

func generatePath(rootPath string, platform, username string) string {
	currentTime := time.Now()
	formattedTime := currentTime.Format("2006-01-02")

	folderName := platform + "_" + username + "_" + formattedTime
	return filepath.Join(rootPath, folderName)
}

func startSegmentMonitoring(done <-chan struct{}, username string, TimeSegment int) {
	logger.Debug("Инициализация горутины с мониторингом записи сегментов", zap.Int("timeSegment", TimeSegment))
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

func watchContext(ctx context.Context, username string, done chan struct{}) {
	logger.Debug("Инициализация горутины со слежкой за контекстом")
	select {
	case <-ctx.Done():
		logger.Info("[" + username + "] Запись стрима прервана по запросу отмены")
		close(done)
		return
	}
}

func runFFmpegCommand(m3u8, path, username string, configEnv *config.Env, done chan struct{}) error {
	logFile := createLogFile(username)
	defer logFile.Close()

	var (
		cmd      *ffmpeg.Stream
		filename string
		args     ffmpeg.KwArgs
	)
	if configEnv.SplitSegments {
		go startSegmentMonitoring(done, username, configEnv.TimeSegment)
		filename = path + "/" + username + "_" + time.Now().Format("15-04-05") + "_%03d.mov"
		args = ffmpeg.KwArgs{
			"c:v":              configEnv.VideoCodec,
			"c:a":              configEnv.AudioCodec,
			"segment_time":     configEnv.TimeSegment,
			"f":                "segment",
			"reset_timestamps": "1",
		}
	} else {
		filename = path + "/" + username + "_" + time.Now().Format("15-04-05") + ".mov"
		args = ffmpeg.KwArgs{
			"c:v": configEnv.VideoCodec,
			"c:a": configEnv.AudioCodec,
		}
	}
	logger.Debug("Инициализация ffmpeg", zap.String("m3u8", m3u8), zap.String("filename", filename), zap.Bool("splitSegments", configEnv.SplitSegments), zap.Any("args", args))
	cmd = ffmpeg.Input(m3u8).Output(filename, args).WithErrorOutput(logFile)

	return cmd.Run()
}

func createLogFile(username string) *os.File {
	logger.Debug("Создание лог-файла ffmpeg", zap.String("path", fmt.Sprintf("logs/"+"ffmpeg-%s.log", username)))
	logFile, err := os.Create(fmt.Sprintf("logs/"+"ffmpeg-%s.log", username))
	if err != nil {
		logger.Warn("Ошибка создания лог-файла ffmpeg", zap.Error(err))
	}
	return logFile
}
