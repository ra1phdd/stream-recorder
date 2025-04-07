package m3u8

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

func (m *M3u8) fetchPlaylist(url string) ([]string, error) {
	resp, err := m.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			m.log.Error("failed to close response body", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		m.log.Error(fmt.Sprintf("[%s/%s] Failed to fetch master playlist", m.sm.Username, m.sm.Platform), nil, slog.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("failed to fetch master playlist with status code %d", resp.StatusCode)
	}

	var segments []string
	var skipCount int

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if skipCount > 0 {
			skipCount--
			continue
		}

		skip, isSegment, segmentURL := m.sl.Platform.ParseM3u8(line, m.sm)
		if isSegment {
			segments = append(segments, segmentURL)
		}
		skipCount = skip
	}

	if err := scanner.Err(); err != nil {
		m.log.Error(fmt.Sprintf("[%s/%s] Buffer scanning error", m.sm.Username, m.sm.Platform), err)
		return nil, err
	}

	return segments, nil
}

func (m *M3u8) downloadSegment(url string) ([]byte, error) {
	const maxAttempts = 10
	var attempt int

	for {
		attempt++
		m.log.Debug(fmt.Sprintf("[%s/%s] Starting download segment", m.sm.Username, m.sm.Platform), slog.String("url", url), slog.Int("attempt", attempt))

		resp, err := m.HTTPClient.Get(url)
		if err != nil {
			if attempt > maxAttempts {
				return nil, fmt.Errorf("reached max attempts (%d) to download segment", maxAttempts)
			}

			m.log.Error(fmt.Sprintf("[%s/%s] Failed to download segment", m.sm.Username, m.sm.Platform), err, slog.String("url", url), slog.Int("attempt", attempt))
			time.Sleep(3 * time.Second * time.Duration(attempt))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			m.log.Error(fmt.Sprintf("[%s/%s] Received non-OK status code while downloading segment", m.sm.Username, m.sm.Platform), nil, slog.String("url", url), slog.Int("status_code", resp.StatusCode), slog.Int("attempt", attempt))
			_ = resp.Body.Close()
			return nil, fmt.Errorf("segment not found (404)")
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			if attempt > maxAttempts {
				return nil, fmt.Errorf("reached max attempts (%d) to download segment", maxAttempts)
			}

			m.log.Error(fmt.Sprintf("[%s/%s] Failed to read segment data", m.sm.Username, m.sm.Platform), err, slog.String("url", url), slog.Int("attempt", attempt))
			_ = resp.Body.Close()
			time.Sleep(3 * time.Second * time.Duration(attempt))
			continue
		}
		_ = resp.Body.Close()

		m.log.Debug(fmt.Sprintf("[%s/%s] Successfully downloaded segment", m.sm.Username, m.sm.Platform), slog.String("url", url), slog.Int("attempt", attempt))
		return data, nil
	}
}
