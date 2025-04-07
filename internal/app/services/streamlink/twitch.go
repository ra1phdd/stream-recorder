package streamlink

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"stream-recorder/internal/app/models"
	"stream-recorder/pkg/logger"
	"strings"
	"time"
)

type TwitchAPI struct {
	log        *logger.Logger
	HTTPClient *http.Client
	ClientID   string
	DeviceID   string
	Headers    map[string]string
}

type IntegrityResponse struct {
	Token      string `json:"token"`
	Expiration int64  `json:"expiration"`
	RequestID  string `json:"request_id"`
}

const (
	UsherURL     = "https://usher.ttvnw.net"
	GqlURL       = "https://gql.twitch.tv/gql"
	IntegrityURL = "https://gql.twitch.tv/integrity"
)

func NewTwitch(log *logger.Logger, clientId, deviceId string) *TwitchAPI {
	return &TwitchAPI{
		log:        log,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
		ClientID:   clientId,
		DeviceID:   deviceId,
	}
}

func (t *TwitchAPI) fetchIntegrity() (string, error) {
	t.log.Debug("Fetching client integrity token", slog.String("url", IntegrityURL))

	req, err := http.NewRequest("POST", IntegrityURL, nil)
	if err != nil {
		t.log.Error("Failed to create request", err)
		return "", err
	}
	req.Header.Add("X-Device-Id", t.DeviceID)
	req.Header.Add("Client-Id", t.ClientID)

	resp, err := t.HTTPClient.Do(req)
	if err != nil {
		t.log.Error("Failed to fetch integrity token", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.log.Error("Failed to read response body", err)
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		t.log.Error("HTTP error in fetchIntegrity", nil, slog.String("response", string(body)), slog.Int("status_code", resp.StatusCode))
		return "", err
	}

	var integrityResp IntegrityResponse
	if err := json.Unmarshal(body, &integrityResp); err != nil {
		t.log.Error("Failed to unmarshal response", err)
		return "", err
	}

	t.log.Debug("Successfully fetched integrity token")
	return integrityResp.Token, nil
}

func (t *TwitchAPI) gqlPersistedQuery(operationName, sha256Hash string, variables map[string]interface{}) interface{} {
	return map[string]interface{}{
		"operationName": operationName,
		"extensions": map[string]interface{}{
			"persistedQuery": map[string]interface{}{
				"version":    1,
				"sha256Hash": sha256Hash,
			},
		},
		"variables": variables,
	}
}

func (t *TwitchAPI) call(data interface{}) (interface{}, error) {
	t.log.Debug("Making TwitchAPI call", slog.String("url", GqlURL))

	ci, err := t.fetchIntegrity()
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.log.Error("Failed to marshal request data", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", GqlURL, bytes.NewBuffer(jsonData))
	if err != nil {
		t.log.Error("Failed to create request", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Client-Id", t.ClientID)
	req.Header.Add("X-Device-Id", t.DeviceID)
	req.Header.Add("Client-Integrity", ci)

	resp, err := t.HTTPClient.Do(req)
	if err != nil {
		t.log.Error("Failed to execute TwitchAPI call", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.log.Error("Failed to read response body", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		t.log.Error("HTTP error in TwitchAPI call", nil, slog.String("response", string(body)), slog.Int("status_code", resp.StatusCode))
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.log.Error("Failed to unmarshal response", err)
		return nil, err
	}

	t.log.Debug("TwitchAPI call successful")
	return result, nil
}

func (t *TwitchAPI) accessToken(channel string) (map[string]interface{}, error) {
	t.log.Debug("Fetching access token", slog.String("channel", channel))

	variables := map[string]interface{}{
		"isLive":     true,
		"login":      channel,
		"isVod":      false,
		"vodID":      "",
		"playerType": "embed",
	}
	query := t.gqlPersistedQuery("PlaybackAccessToken", "0828119ded1c13477966434e15800ff57ddacf13ba1911c129dc2200705b0712", variables)

	response, err := t.call(query)
	if err != nil {
		t.log.Error("Failed to get access token", nil, slog.String("channel", channel), err)
		return nil, err
	}

	results, ok := response.(map[string]interface{})
	if !ok {
		t.log.Error("Unexpected response format for access token", nil, slog.Any("response", response))
		return nil, errors.New("unexpected response format")
	}

	data, ok := results["data"].(map[string]interface{})
	if !ok {
		t.log.Error("Data not found in response", nil, slog.Any("response", results))
		return nil, errors.New("data not found in response")
	}

	streamToken, ok := data["streamPlaybackAccessToken"].(map[string]interface{})
	if !ok {
		t.log.Error("streamPlaybackAccessToken not found", nil, slog.Any("response", data))
		return nil, errors.New("streamPlaybackAccessToken not found")
	}

	t.log.Debug("Access token fetched successfully", slog.String("channel", channel))
	return map[string]interface{}{
		"signature": streamToken["signature"],
		"value":     streamToken["value"],
	}, nil
}

func (t *TwitchAPI) GetMasterPlaylist(channel string) (string, error) {
	accessToken, err := t.accessToken(channel)
	if err != nil {
		t.log.Error("Failed to get access token", nil, slog.String("channel", channel), err)
		return "", err
	}

	return fmt.Sprintf("%s/api/channel/hls/%s.m3u8?player=twitchweb&platform=web&supported_codecs=h265,h264&p=715347&type=any&allow_source=true&allow_audio_only=true&allow_spectre=false&sig=%s&token=%s", UsherURL, channel, accessToken["signature"].(string), url.QueryEscape(accessToken["value"].(string))), nil
}

func (t *TwitchAPI) FindMediaPlaylist(masterPlaylist, quality string) (string, error) {
	resp, err := http.Get(masterPlaylist)
	if err != nil {
		t.log.Error("Failed to get master playlist", nil, slog.String("masterPlaylist", masterPlaylist), err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode != http.StatusNotFound {
			t.log.Error("HTTP error in find media playlist", nil, slog.Int("status_code", resp.StatusCode))
		}

		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var resolution string
	resUri := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#EXT-X-STREAM-INF") {
			t.log.Debug("Found tag #EXT-X-STREAM-INF", slog.String("line", line))

			if resStart := strings.Index(line, "RESOLUTION="); resStart != -1 {
				resEnd := strings.Index(line[resStart:], ",")
				if resEnd == -1 {
					resEnd = len(line)
				} else {
					resEnd += resStart
				}
				resolution = line[resStart+10 : resEnd]
			}
			continue
		}

		if strings.HasPrefix(line, "http") {
			t.log.Debug("Found URL resolution", slog.String("line", line))
			resUri[resolution] = line
		}
	}

	if err := scanner.Err(); err != nil {
		t.log.Error("Buffer scanning error", err)
		return "", err
	}

	needUri, err := t.FindNeedQuality(resUri, quality)
	if err != nil {
		t.log.Error("Failed to find need quality", err, slog.String("quality", quality))
		return "", err
	}

	return needUri, nil
}

func (t *TwitchAPI) FindNeedQuality(resUri map[string]string, quality string) (string, error) {
	var needResolution string
	switch quality {
	case "1440p":
		needResolution = "=2560x1440"
	case "1080p":
		needResolution = "=1920x1080"
	case "720p":
		needResolution = "=1280x720"
	case "480p":
		needResolution = "=852x480"
	case "360p":
		needResolution = "=640x360"
	case "160p":
		needResolution = "=284x160"
	case "best":
		var bestResolution string
		for res := range resUri {
			if bestResolution == "" || t.compareResolutions(res, bestResolution) == 1 {
				bestResolution = res
			}
		}
		if bestResolution == "" {
			return "", errors.New("no streams available")
		}
		return resUri[bestResolution], nil
	default:
		return "", errors.New("quality not supported")
	}

	for res, uri := range resUri {
		if res == needResolution {
			return uri, nil
		}
	}

	return "", fmt.Errorf("quality %s not found", quality)
}

func (t *TwitchAPI) ParseM3u8(line string, m *models.StreamMetadata) (skipCount int, isSegment bool, segmentURL string) {
	if strings.HasPrefix(line, "#EXT-X-TARGETDURATION") && !*m.SkipTargetDuration {
		t.log.Debug(fmt.Sprintf("[%s/%s] Found tag #EXT-X-TARGETDURATION", m.Username, m.Platform), slog.String("line", line))

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			t.log.Error(fmt.Sprintf("[%s/%s] Failed to parse target duration", m.Username, m.Platform), nil, slog.String("line", line))
			return
		}

		parsedTime, err := strconv.Atoi(parts[1])
		if err != nil {
			t.log.Error(fmt.Sprintf("[%s/%s] Failed to parse tag #EXT-X-TARGETDURATION", m.Username, m.Platform), nil, slog.String("line", line), slog.Any("parts", parts))
			return
		}

		*m.WaitingTime = time.Duration(parsedTime) * time.Second
		*m.SkipTargetDuration = true
		return
	}

	if strings.HasPrefix(line, "#EXT-X-TWITCH-TOTAL-SECS") {
		t.log.Debug(fmt.Sprintf("[%s/%s] Found tag #EXT-X-TWITCH-TOTAL-SECS", m.Username, m.Platform), slog.String("line", line))

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			t.log.Error(fmt.Sprintf("[%s/%s] Failed to split total seconds in parts", m.Username, m.Platform), nil, slog.String("line", line), slog.Any("parts", parts))
			return
		}

		timeParts := strings.Split(parts[1], ".")
		if len(timeParts) != 2 {
			t.log.Error(fmt.Sprintf("[%s/%s] Failed to split total seconds in timeParts", m.Username, m.Platform), nil, slog.String("line", line), slog.Any("timeParts", timeParts))
			return
		}

		parsedTime, err := strconv.Atoi(timeParts[0])
		if err != nil {
			t.log.Error(fmt.Sprintf("[%s/%s] Error converting timeParts[0] to a number", m.Username, m.Platform), nil, slog.String("line", line), slog.Any("timeParts", timeParts), err)
			return
		}

		*m.TotalDurationStream = time.Duration(parsedTime) * time.Second
		if *m.StartDurationStream == 0 {
			m.StartDurationStream = m.TotalDurationStream
		}
		return
	}

	if strings.HasPrefix(line, "#EXTINF") && strings.Contains(line, `Amazon`) {
		t.log.Debug(fmt.Sprintf("[%s/%s] Found ad tag 'Amazon'", m.Username, m.Platform), slog.String("line", line))
		skipCount = 1
		return
	}

	if !strings.HasPrefix(line, "#") {
		isSegment = true
		segmentURL = line
	}
	return
}

// Функция для сравнения разрешений (возвращает 1, если res1 больше res2, -1 если меньше, 0 если равны)
func (t *TwitchAPI) compareResolutions(res1, res2 string) int {
	parseResolution := func(res string) (int, int) {
		parts := strings.Split(res, "x")
		if len(parts) != 2 {
			return 0, 0
		}
		width, _ := strconv.Atoi(parts[0])
		height, _ := strconv.Atoi(parts[1])
		return width, height
	}

	w1, h1 := parseResolution(res1)
	w2, h2 := parseResolution(res2)

	if w1 > w2 || (w1 == w2 && h1 > h2) {
		return 1
	} else if w1 < w2 || (w1 == w2 && h1 < h2) {
		return -1
	}
	return 0
}
