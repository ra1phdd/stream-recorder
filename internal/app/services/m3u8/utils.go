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

func (m *M3u8) generateFilePaths() (fileSegments, pathFileWithoutExt string) {
	dirName := fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, m.currentDate)
	fileName := fmt.Sprintf("%s_%s_%s", m.sm.Platform, m.sm.Username, m.formatDuration(*m.sm.StartDurationStream))

	fileSegments = filepath.Join(m.c.TempPATH, dirName, fileName+".txt")
	pathFileWithoutExt = filepath.Join(m.c.MediaPATH, dirName, fileName)

	return fileSegments, pathFileWithoutExt
}

func (m *M3u8) formatDuration(d time.Duration) string {
	hours := d / time.Hour
	d -= hours * time.Hour
	mins := d / time.Minute
	d -= mins * time.Minute
	secs := d / time.Second
	return fmt.Sprintf("%dh%dm%ds", hours, mins, secs)
}

func (m *M3u8) ChangeIsNeedCut(value bool) {
	m.isNeedCut = value
}

func (m *M3u8) ChangeIsCancel(value bool) {
	m.isCancel = value
}
