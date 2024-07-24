//go:build windows && amd64

package embed

import (
	"embed"
	"fmt"
)

//go:embed bin/streamlink_windows_x64.exe bin/ffmpeg_windows.exe
var fs embed.FS

func getFileName(name string) (string, error) {
	switch name {
	case "streamlink":
		return "streamlink_windows_x64.exe", nil
	case "ffmpeg":
		return "ffmpeg_windows.exe", nil
	}
	return "", fmt.Errorf("unknown file: %s", name)
}
