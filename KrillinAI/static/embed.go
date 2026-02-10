package static

import "embed"

//go:embed index.html background.jpg source/*.svg source/*.png js/*.js css/*.css
var EmbeddedFiles embed.FS
