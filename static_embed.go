package static

import (
	"embed"
	_ "embed"
)

//go:embed static
var StaticFiles embed.FS
