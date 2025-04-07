package m3u8

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"stream-recorder/pkg/ffmpeg"
	"sync"
)

func (m *M3u8) processSegments(segments []string, baseDir string) {
	var wg sync.WaitGroup
	dataMap := make(map[int]string)

	for index, segment := range segments {
		url := m.getShortFileName(segment)
		if m.rottenDownloadedSegments.Has(url) {
			continue
		}
		dataMap[index] = url

		wg.Add(1)
		go func(index int, segment string, url string) {
			defer wg.Done()

			data, err := m.downloadSegment(segment)
			if err != nil || len(data) == 0 {
				dataMap[index] = ""
				m.log.Error(fmt.Sprintf("[%s/%s] Error downloading segment", m.sm.Username, m.sm.Platform), err, slog.String("segmentURL", segment))
				return
			}

			tsPath := filepath.Join(baseDir, url+".ts")
			filePath := filepath.Join(baseDir, fmt.Sprintf("%s.%s", url, m.c.FileFormat))

			if err := os.WriteFile(tsPath, data, 0644); err != nil {
				dataMap[index] = ""
				m.log.Error(fmt.Sprintf("[%s/%s] Failed to write segment to file", m.sm.Username, m.sm.Platform), err, slog.String("filePath", tsPath))
				return
			}

			segmentFFmpeg, err := ffmpeg.NewFfmpeg(m.c.FFmpegPATH)
			if err != nil {
				dataMap[index] = ""
				m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
				os.Remove(tsPath)
				return
			}

			err = segmentFFmpeg.Yes().
				LogLevel("error").
				VideoCodec(m.c.VideoCodec).
				AudioCodec(m.c.AudioCodec).
				Execute(tsPath, filePath)
			if err != nil {
				dataMap[index] = ""
				m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
			}

			if err := os.Remove(tsPath); err != nil {
				m.log.Error(fmt.Sprintf("[%s/%s] Failed to remove temp file", m.sm.Username, m.sm.Platform), err)
			}
		}(index, segment, url)
	}
	wg.Wait()

	for _, data := range dataMap {
		m.rottenDownloadedSegments.Add(data)
		m.downloadedSegments.Add(data)
	}
}
