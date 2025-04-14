package scheduler

import (
	"os"
	"path/filepath"
	"stream-recorder/internal/app/services/m3u8"
	"strings"
)

func (s *Scheduler) Recovery() {
	s.log.Warn("Recovering streams...")
	var files = make(map[string][]string)
	var txtFiles = make(map[string][]string)

	err := filepath.Walk(s.cfg.TempPATH, func(path string, info os.FileInfo, err error) error {
		if info.Size() == 0 {
			os.Remove(path)
		}

		if info.IsDir() || info.Name() == "ffmpeg" {
			return nil
		}

		if strings.HasSuffix(info.Name(), "_video.txt") {
			txtFiles[filepath.Dir(path)] = append(txtFiles[filepath.Dir(path)], info.Name())
			return nil
		}

		files[filepath.Dir(path)] = append(files[filepath.Dir(path)], info.Name())
		return nil
	})
	if err != nil {
		s.log.Error("Error reading temp path", err)
		return
	}

	for path, f := range txtFiles {
		for _, file := range f {
			tempPath := filepath.Join(s.cfg.TempPATH, filepath.Base(path), strings.TrimSuffix(file, "_video.txt"))
			mediaPath := filepath.Join(s.cfg.MediaPATH, filepath.Base(path), strings.TrimSuffix(file, "_video.txt"))

			go func(tempPath, mediaPath string) {
				m, err := m3u8.New(s.log, "", "", false, 0, s.cfg, s.u)
				if err != nil {
					s.log.Error("Error creating m3u8", err)
					return
				}

				m.ConcatAndCleanup(tempPath, mediaPath)
			}(tempPath, mediaPath)
		}
	}

	//for path := range files {
	//	hash, _ := s.u.RandomToken(32, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	//	tempPath := filepath.Join(s.cfg.TempPATH, filepath.Base(path), s.u.RemoveDateFromPath(filepath.Base(path))+"_"+hash+"_recovery")
	//	mediaPath := filepath.Join(s.cfg.MediaPATH, filepath.Base(path), s.u.RemoveDateFromPath(filepath.Base(path))+"_"+hash+"_recovery")
	//
	//	go func(tempPath, mediaPath string) {
	//		m, err := m3u8.New(s.log, "", "", false, 0, s.cfg, s.u)
	//		if err != nil {
	//			s.log.Error("Error creating m3u8", err)
	//			return
	//		}
	//
	//		err = m.FlushTxtToDisk(tempPath)
	//		if err != nil {
	//			s.log.Error("Error flush txt to disk", err)
	//			return
	//		}
	//		m.ConcatAndCleanup(tempPath, mediaPath)
	//	}(tempPath, mediaPath)
	//}

	files = make(map[string][]string)
	txtFiles = make(map[string][]string)
	s.log.Warn("Successfully recovered streams")
}
