package assets

import "embed"

//go:embed frontend/dist
var assets embed.FS

func Get() embed.FS {
	return assets
}
