package locale

import "io/fs"

type FileSystem struct {
	fs.FS
}

func New(fs fs.FS) FileSystem {
	return FileSystem{FS: fs}
}
