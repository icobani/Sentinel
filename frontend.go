package sentinel

import "embed"

//go:embed all:dist/browser
var FrontendFS embed.FS
