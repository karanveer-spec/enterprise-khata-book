package khatabook

import "embed"

//go:embed all:web
var EmbeddedWebFS embed.FS
