//go:build darwin && amd64

package embed

import (
	"embed"
)

//go:embed ffmpeg/ffmpeg-darwin-amd64
var fsFfmpeg embed.FS

func getFileFfmpeg() string {
	return "ffmpeg-darwin-amd64"
}
