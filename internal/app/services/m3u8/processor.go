package m3u8

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"stream-recorder/pkg/ffmpeg"
	"sync"
)

func (m *M3u8) processSegments(segments []string, baseDir string) bool {
	if len(segments) == 0 {
		return false
	}

	var wg sync.WaitGroup
	var dataMap = make([]string, len(segments))

	for index, segment := range segments {
		url := m.getShortFileName(segment)
		if m.rottenDownloadedSegments.Has(url) || m.downloadedSegments.Has(url) {
			continue
		}
		dataMap[index] = url

		wg.Add(1)
		go func(index int, segment string, url string) {
			defer wg.Done()

			data, err := m.downloadSegment(segment)
			if err != nil || len(data) == 0 {
				dataMap[index] = "none"
				m.log.Error(fmt.Sprintf("[%s/%s] Error downloading segment", m.sm.Username, m.sm.Platform), err, slog.String("segmentURL", segment))
				return
			}

			tsPath := filepath.Join(baseDir, fmt.Sprintf("%s_temp.ts", url))
			videoPath := filepath.Join(baseDir, fmt.Sprintf("%s.ts", url))
			audioPath := filepath.Join(baseDir, fmt.Sprintf("%s.wav", url))

			if err := os.WriteFile(tsPath, data, 0644); err != nil {
				dataMap[index] = "none"
				m.log.Error(fmt.Sprintf("[%s/%s] Failed to write segment to file", m.sm.Username, m.sm.Platform), err, slog.String("filePath", tsPath))
				return
			}

			segmentFFmpeg, err := ffmpeg.NewFfmpeg(m.c.FFmpegPATH)
			if err != nil {
				dataMap[index] = "none"
				m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
				os.Remove(tsPath)
				return
			}

			err = segmentFFmpeg.Yes().
				LogLevel("error").
				Format("mpegts").
				VideoCodec(m.c.VideoCodec).
				AudioCodec("none").
				ExtraArgs([]string{"-copyts"}).
				Execute([]string{tsPath}, videoPath)
			if err != nil {
				dataMap[index] = "none"
				m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
			}

			segmentFFmpeg.Clear()

			err = segmentFFmpeg.Yes().
				LogLevel("error").
				VideoCodec("none").
				AudioCodec(m.c.AudioCodec).
				Execute([]string{tsPath}, audioPath)
			if err != nil {
				dataMap[index] = "none"
				m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
			}

			if err := os.Remove(tsPath); err != nil {
				m.log.Error(fmt.Sprintf("[%s/%s] Failed to remove temp file", m.sm.Username, m.sm.Platform), err)
			}
		}(index, segment, url)
	}
	wg.Wait()

	var isErrDownload bool
	for _, data := range dataMap {
		if data == "" {
			continue
		}

		if data == "none" {
			isErrDownload = true
			break
		}

		m.downloadedSegments.Add(data)
	}

	return isErrDownload
}
