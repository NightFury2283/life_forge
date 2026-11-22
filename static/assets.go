package webassets

import "embed"

//go:embed app.html
var Page []byte

//go:embed app.js planner.js workspace.css theme.js theme-light.css
var Files embed.FS
