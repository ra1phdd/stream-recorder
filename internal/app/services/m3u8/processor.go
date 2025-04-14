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
	var dataMap = make([][]byte, len(segments))
	var urlMap = make([]string, len(segments))

	for index, segment := range segments {
		url := m.u.GetShortFileName(segment)
		if m.downloadedSegments.Has(url) {
			urlMap[index] = ""
			continue
		}
		urlMap[index] = url

		wg.Add(1)
		go func(index int, segment string) {
			defer wg.Done()

			data, err := m.downloadSegment(segment)
			if err != nil || len(data) == 0 {
				m.log.Error(fmt.Sprintf("[%s/%s] Error downloading segment", m.sm.Username, m.sm.Platform), err, slog.String("segmentURL", segment))
				return
			}
			dataMap[index] = data
		}(index, segment)
	}
	wg.Wait()

	var isErrDownload bool
	for i, url := range urlMap {
		if url == "" {
			continue
		}

		if len(dataMap[i]) == 0 {
			if err := m.u.CreateDirectoryIfNotExist(filepath.Join(m.c.TempPATH, m.streamDir)); err != nil {
				m.log.Error(fmt.Sprintf("[%s/%s] Failed create temp directory", m.sm.Username, m.sm.Platform), err)
			}

			err := m.flushSegmentToDisk(baseDir, url)
			if err == nil {
				m.segmentId++
				m.dataSegments = m.dataSegments[:0]
			}

			isErrDownload = true
			break
		}

		m.dataSegments = append(m.dataSegments, dataMap[i]...)
		if len(m.dataSegments) >= m.c.BufferSize || m.GetIsNeedCut() || m.GetIsCancel() {
			err := m.flushSegmentToDisk(baseDir, url)
			if err != nil {
				isErrDownload = true
				break
			}
			m.segmentId++
			m.dataSegments = m.dataSegments[:0]
		}

		m.downloadedSegments.Add(url)
	}

	return isErrDownload
}

func (m *M3u8) flushSegmentToDisk(baseDir, url string) error {
	tsPath := filepath.Join(baseDir, fmt.Sprintf("%d_%s_temp.ts", m.segmentId, url))
	videoPath := filepath.Join(baseDir, fmt.Sprintf("%d_%s.%s", m.segmentId, url, m.c.FileFormat))
	audioPath := filepath.Join(baseDir, fmt.Sprintf("%d_%s.%s", m.segmentId, url, m.getRecommendedAudioFormat(m.c.AudioCodec)))

	if err := os.WriteFile(tsPath, m.dataSegments, 0644); err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Failed to write segment to file", m.sm.Username, m.sm.Platform), err, slog.String("filePath", tsPath))
		return err
	}

	segmentFFmpeg, err := ffmpeg.NewFfmpeg(m.c.FFmpegPATH)
	if err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
		os.Remove(tsPath)
		return err
	}

	err = segmentFFmpeg.Yes().
		LogLevel("error").
		VideoCodec(m.c.VideoCodec).
		AudioCodec("none").
		Execute([]string{tsPath}, videoPath)
	if err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
	}

	segmentFFmpeg.Clear()

	err = segmentFFmpeg.Yes().
		LogLevel("error").
		VideoCodec("none").
		AudioCodec(m.c.AudioCodec).
		Execute([]string{tsPath}, audioPath)
	if err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Failed run ffmpeg", m.sm.Username, m.sm.Platform), err)
	}

	if err := os.Remove(tsPath); err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Failed to remove temp file", m.sm.Username, m.sm.Platform), err)
	}

	return nil
}
