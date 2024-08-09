package recorder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"stream-recorder/config"
	"stream-recorder/pkg/embed"
	"stream-recorder/pkg/logger"
	"strings"
	"time"

	"go.uber.org/zap"
)

func Init(ctx context.Context, configEnv *config.Env, platform string, username string, quality string) {
	logger.Debug("Инициализация горутины проверки наличия и записи стрима", zap.String("platform", platform), zap.String("username", username), zap.String("quality", quality))
	done := make(chan struct{})
	defer close(done)

	go watchContext(ctx, username, platform, done)

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
			logger.Fatalf("Ошибка создания папки для записи стрима", username, platform)
		}

		logger.Infof("Начинаю запись стрима...", username, platform)

		err = runFFmpegCommand(m3u8, path, username, platform, configEnv, done)
		if err != nil {
			logger.Errorf("Ошибка выполнения команды FFmpeg", username, platform, zap.Error(err))
		} else {
			logger.Infof("Запись стрима окончена. Возвращаюсь к проверке наличия стрима...", username, platform)
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
		logger.Fatalf("Неизвестная платформа", username, platform, zap.String("platform", platform))
	}
	logger.Debug("Полученная ссылка стрима", zap.String("link", link))

	for {
		cmd := exec.Command(embed.GetTempFileName("streamlink"), "--stream-url", link, quality)

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
		case <-done:
			stdout := stdoutBuf.String()

			if strings.HasPrefix(stdout, "https://") {
				return stdout
			} else if strings.Contains(stdout, "No playable streams found on this URL") {
				logger.Infof("Стример не стримит, повторная проверка через "+fmt.Sprint(timeCheck)+" секунд", username, platform)
				time.Sleep(time.Duration(timeCheck) * time.Second)
				continue
			} else if stdout == "" {
				logger.Infof("Получение URL занимает больше времени, чем предполагалось... Ожидаю ответа", username, platform)

				for {
					time.Sleep(1 * time.Second)

					stdout = stdoutBuf.String()
					if strings.HasPrefix(stdout, "https://") {
						return stdout
					} else if strings.Contains(stdout, "No playable streams found on this URL") {
						logger.Infof("Стример не стримит, повторная проверка через "+fmt.Sprint(timeCheck)+" секунд", username, platform)
						time.Sleep(time.Duration(timeCheck) * time.Second)
						break
					} else if stdout != "" {
						logger.Errorf("Неизвестная ошибка при получении URL потока", username, platform, zap.Any("stdout", stdout))
						time.Sleep(time.Duration(timeCheck) * time.Second)
						break
					}
				}
			} else {
				logger.Errorf("Неизвестная ошибка при получении URL потока", username, platform, zap.Any("stdout", stdout))
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

func startSegmentMonitoring(done <-chan struct{}, username, platform string, TimeSegment int) {
	logger.Debug("Инициализация горутины с мониторингом записи сегментов", zap.Int("timeSegment", TimeSegment))
	time.Sleep(time.Duration(TimeSegment) * time.Second)
	for {
		select {
		case <-done:
			logger.Infof("Запись стрима завершена", username, platform)
			return
		case <-time.After(time.Duration(TimeSegment) * time.Second):
			logger.Infof("Сегмент записан, перехожу к следующему", username, platform)
		}
	}
}

func watchContext(ctx context.Context, username, platform string, done chan struct{}) {
	logger.Debug("Инициализация горутины со слежкой за контекстом")
	select {
	case <-ctx.Done():
		logger.Infof("Запись стрима прервана по запросу отмены", username, platform)
		close(done)
		return
	}
}

func runFFmpegCommand(m3u8, path, username, platform string, configEnv *config.Env, done chan struct{}) error {
	logFile := createLogFile(username)
	defer logFile.Close()

	var (
		filename string
		args     []string
	)
	if configEnv.SplitSegments {
		go startSegmentMonitoring(done, username, platform, configEnv.TimeSegment)
		filename = fmt.Sprintf("%s/%s_%s_%%03d.mov", path, username, time.Now().Format("15-04-05"))
		args = []string{
			"-re",
			"-protocol_whitelist", "file,crypto,data,http,https,tls,tcp",
			"-loglevel", "warning",
			"-i", strings.Trim(m3u8, "\n"),
			"-async", "1",
			"-fps_mode", "cfr",
			"-fflags", "+genpts",
			"-bufsize", "20M",
			"-reconnect", "1",
			"-reconnect_at_eof", "1",
			"-reconnect_streamed", "1",
			"-reconnect_delay_max", "2",
			"-c:v", configEnv.VideoCodec,
			"-c:a", configEnv.AudioCodec,
			"-segment_time", fmt.Sprint(configEnv.TimeSegment),
			"-f", "segment",
			"-reset_timestamps", "1",
			"'" + filename + "'",
		}
	} else {
		filename = fmt.Sprintf("%s/%s_%s.mov", path, username, time.Now().Format("15-04-05"))
		args = []string{
			"-re",
			"-protocol_whitelist", "file,crypto,data,http,https,tls,tcp",
			"-loglevel", "warning",
			"-i", strings.Trim(m3u8, "\n"),
			"-async", "1",
			"-fps_mode", "cfr",
			"-fflags", "+genpts",
			"-bufsize", "20M",
			"-reconnect", "1",
			"-reconnect_at_eof", "1",
			"-reconnect_streamed", "1",
			"-reconnect_delay_max", "2",
			"-c:v", configEnv.VideoCodec,
			"-c:a", configEnv.AudioCodec,
			"'" + filename + "'",
		}
	}
	logger.Debug("Инициализация ffmpeg", zap.String("m3u8", m3u8), zap.String("filename", filename), zap.Bool("splitSegments", configEnv.SplitSegments), zap.Any("args", args))

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", "ffmpeg "+strings.Join(args, " "))
	case "darwin", "linux":
		cmd = exec.Command("sh", "-c", "ffmpeg "+strings.Join(args, " "))
	default:
		return fmt.Errorf("неподдерживаемая ОС: %s", runtime.GOOS)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = logFile

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ошибка выполнения команды ffmpeg: %w", err)
	}

	return nil
}

func createLogFile(username string) *os.File {
	logger.Debug("Создание лог-файла ffmpeg", zap.String("path", fmt.Sprintf("logs/"+"ffmpeg-%s.log", username)))
	logFile, err := os.Create(fmt.Sprintf("logs/"+"ffmpeg-%s.log", username))
	if err != nil {
		logger.Warn("Ошибка создания лог-файла ffmpeg", zap.Error(err))
	}
	return logFile
}
