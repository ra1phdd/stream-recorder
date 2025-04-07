//go:build darwin && arm64

package ffmpeg

import (
	"embed"
)

//go:embed bin/ffmpeg-darwin-arm64
var ffmpeg embed.FS
