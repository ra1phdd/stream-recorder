//go:build darwin && amd64

package ffmpeg

import (
	"embed"
)

//go:embed bin/ffmpeg-darwin-amd64
var ffmpeg embed.FS
