//go:build windows && amd64

package ffmpeg

import (
	"embed"
)

//go:embed bin/ffmpeg-windows-amd64.exe
var ffmpeg embed.FS
