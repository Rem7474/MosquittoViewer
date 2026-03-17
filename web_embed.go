package mosquittoviewer

import "embed"

// WebFS embeds the frontend build output from /web.
//go:embed all:web
var WebFS embed.FS
