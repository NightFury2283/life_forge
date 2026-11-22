package webassets

import "embed"

//go:embed app.html
var Page []byte

//go:embed app.css app.js
var Files embed.FS
