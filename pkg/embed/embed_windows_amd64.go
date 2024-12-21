//go:build windows && amd64

package embed

import (
	"embed"
)

//go:embed ffmpeg/ffmpeg-windows-amd64.exe
var fsFfmpeg embed.FS

func getFileFfmpeg() string {
	return "ffmpeg-windows-amd64.exe"
}
