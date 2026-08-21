//go:build !dist

package webdist

import "io/fs"

const Embedded = false

func App() fs.FS     { return nil }
func Boolean() fs.FS { return nil }
