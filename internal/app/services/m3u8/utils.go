package m3u8

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (m *M3u8) generateFilePaths(streamDir string) (string, string) {
	fileName := fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, m.u.FormatDuration(*m.sm.StartDurationStream))

	return filepath.Join(m.c.TempPATH, streamDir, fileName), filepath.Join(m.c.MediaPATH, streamDir, fileName)
}

func (m *M3u8) getRecommendedAudioFormat(codec string) string {
	codec = strings.ToLower(strings.TrimSpace(codec))

	switch codec {
	case "mp3", "libmp3lame":
		return "mp3"
	case "aac", "libfdk_aac", "aac_latm":
		return "aac"
	case "flac":
		return "flac"
	case "vorbis", "libvorbis":
		return "ogg"
	case "opus", "libopus":
		return "opus"
	case "pcm_s16le", "pcm_s24le", "pcm_s32le":
		return "wav"
	default:
		return "aac"
	}
}

func (m *M3u8) GetIsNeedCut() bool {
	m.muCut.Lock()
	defer m.muCut.Unlock()

	return m.isNeedCut
}

func (m *M3u8) ChangeIsNeedCut(value bool) {
	m.muCut.Lock()
	defer m.muCut.Unlock()

	m.isNeedCut = value
}

func (m *M3u8) GetIsCancel() bool {
	m.muCancel.Lock()
	defer m.muCancel.Unlock()

	return m.isCancel
}

func (m *M3u8) ChangeIsCancel(value bool) {
	m.muCancel.Lock()
	defer m.muCancel.Unlock()

	m.isCancel = value
}
