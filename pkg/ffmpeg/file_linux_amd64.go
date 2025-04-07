//go:build linux && amd64

package ffmpeg

import (
	"embed"
)

//go:embed bin/ffmpeg-linux-amd64
var ffmpeg embed.FS
