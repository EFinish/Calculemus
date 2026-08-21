//go:build dist

// Package webdist bakes the built web apps into the server binary so a
// release is one self-contained file. It lives at the repo root because
// go:embed can only reference paths at or below its own directory, and the
// dist folders belong to two different apps. Without the "dist" build tag
// the stub in webdist_stub.go compiles instead, so ordinary builds and
// `go test ./...` never require the apps to be built first.
package webdist

import (
	"embed"
	"io/fs"
)

//go:embed all:app/dist all:app-boolean/dist
var bundled embed.FS

const Embedded = true

// App is the relational workbench (served at /).
func App() fs.FS { return mustSub("app/dist") }

// Boolean is the frozen pre-M6 edition (served at /boolean/).
func Boolean() fs.FS { return mustSub("app-boolean/dist") }

func mustSub(dir string) fs.FS {
	sub, err := fs.Sub(bundled, dir)
	if err != nil {
		panic(err) // paths are fixed at compile time; failure is a build bug
	}
	return sub
}
