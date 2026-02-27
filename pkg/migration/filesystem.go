package migration

import "io/fs"

type FileSystem struct {
	fs.FS
}

func New(f fs.FS) FileSystem {
	return FileSystem{FS: f}
}
