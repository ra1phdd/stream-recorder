//go:build linux && arm64

package embed

import (
	"embed"
	"fmt"
)

//go:embed bin/streamlink bin/ffmpeg_linux_arm64
var fs embed.FS

func getFileName(name string) (string, error) {
	switch name {
	case "streamlink":
		return "streamlink", nil
	case "ffmpeg":
		return "ffmpeg_linux_arm64", nil
	}
	return "", fmt.Errorf("unknown file: %s", name)
}
