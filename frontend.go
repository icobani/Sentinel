package sentinel

import "embed"

//go:embed all:frontend/dist/browser
var FrontendFS embed.FS
