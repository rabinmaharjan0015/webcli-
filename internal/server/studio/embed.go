package studio

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed index.html
var studioFiles embed.FS

func Handler() http.Handler {
	sub, err := fs.Sub(studioFiles, ".")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
