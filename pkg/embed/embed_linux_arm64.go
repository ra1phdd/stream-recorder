//go:build linux && arm64

package embed

import (
	"embed"
)

//go:embed ffmpeg/ffmpeg-linux-arm64
var fsFfmpeg embed.FS

func getFileFfmpeg() string {
	return "ffmpeg-linux-arm64"
}
