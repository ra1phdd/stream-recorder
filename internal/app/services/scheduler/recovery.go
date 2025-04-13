package scheduler

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"stream-recorder/internal/app/services/m3u8"
	"strings"
)

func (s *Scheduler) Recovery() {
	s.log.Warn("Recovering streams...")
	var files = make(map[string][]string)

	err := filepath.Walk(s.cfg.TempPATH, func(path string, info os.FileInfo, err error) error {
		if info.Size() == 0 {
			os.Remove(path)
		}

		if info.IsDir() || !strings.Contains(info.Name(), ".wav") && !strings.Contains(info.Name(), ".ts") {
			return nil
		}

		files[path] = append(files[path], info.Name())

		return nil
	})
	if err != nil {
		s.log.Error("Error reading temp path", err)
		return
	}

	for path, f := range files {
		tempPath := filepath.Join(s.cfg.TempPATH, filepath.Base(filepath.Dir(path)), filepath.Base(filepath.Dir(path)))
		mediaPath := filepath.Join(s.cfg.MediaPATH, filepath.Base(filepath.Dir(path)), filepath.Base(filepath.Dir(path)))

		sort.Slice(f, func(i, j int) bool {
			return extractNumber(f[i]) < extractNumber(f[j])
		})

		if len(f) > 0 {
			f = f[:len(f)-1]
		}

		m, err := m3u8.New(s.log, "", "", false, 0, s.cfg, s.u)
		if err != nil {
			s.log.Error("Error creating m3u8", err)
			continue
		}

		err = m.FlushTxtToDisk(tempPath)
		if err != nil {
			s.log.Error("Error flush txt to disk", err)
			continue
		}
		m.ConcatAndCleanup(tempPath, mediaPath)
	}

	s.log.Warn("Successfully recovered streams")
}

func extractNumber(filename string) int {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 1 {
		return -1
	}
	num, _ := strconv.Atoi(parts[0])
	return num
}
