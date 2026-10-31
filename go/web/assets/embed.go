package assets

import (
	"embed"
	"io/fs"
)

// FS contains the embedded static web assets built into the AIA binary.
//
//go:embed dist
var FS embed.FS

// Sub returns an fs.FS rooted at the "dist" directory for clean static serving.
func Sub() fs.FS {
	sub, err := fs.Sub(FS, "dist")
	if err != nil {
		panic("failed to get dist sub filesystem: " + err.Error())
	}
	return sub
}
