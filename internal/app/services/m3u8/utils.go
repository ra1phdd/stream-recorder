package m3u8

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"
)

func (m *M3u8) getShortFileName(url string) string {
	hasher := md5.New()
	hasher.Write([]byte(url))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (m *M3u8) generateFilePaths(streamDir string) (string, string) {
	fileName := fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, m.formatDuration(*m.sm.StartDurationStream))

	return filepath.Join(m.c.TempPATH, streamDir, fileName), filepath.Join(m.c.MediaPATH, streamDir, fileName)
}

func (m *M3u8) formatDuration(d time.Duration) string {
	hours := d / time.Hour
	d -= hours * time.Hour
	mins := d / time.Minute
	d -= mins * time.Minute
	secs := d / time.Second
	return fmt.Sprintf("%dh%dm%ds", hours, mins, secs)
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
