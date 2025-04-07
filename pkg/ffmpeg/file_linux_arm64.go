//go:build linux && arm64

package ffmpeg

import (
	"embed"
)

//go:embed bin/ffmpeg-linux-arm64
var ffmpeg embed.FS
