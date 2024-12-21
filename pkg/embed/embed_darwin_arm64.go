//go:build darwin && arm64

package embed

import (
	"embed"
)

//go:embed ffmpeg/ffmpeg-darwin-arm64
var fsFfmpeg embed.FS

func getFileFfmpeg() string {
	return "ffmpeg-darwin-arm64"
}
