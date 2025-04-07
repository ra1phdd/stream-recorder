//go:build windows && arm64

package ffmpeg

import (
	"embed"
)

//go:embed bin/ffmpeg-windows-arm64.exe
var ffmpeg embed.FS
