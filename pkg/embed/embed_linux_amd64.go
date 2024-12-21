//go:build linux && amd64

package embed

import (
	"embed"
)

//go:embed ffmpeg/ffmpeg-linux-amd64
var fsFfmpeg embed.FS

func getFileFfmpeg() string {
	return "ffmpeg-linux-amd64"
}
