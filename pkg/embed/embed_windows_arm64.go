//go:build windows && arm64

package embed

import (
	"embed"
)

//go:embed ffmpeg/ffmpeg-windows-arm64.exe
var fsFfmpeg embed.FS

func getFileFfmpeg() string {
	return "ffmpeg-windows-arm64.exe"
}
