package io

import (
	"io"
	"io/fs"
)

// CreateFS defines a file system interface that supports creating files and directories.
// It extends basic file system operations with write capabilities for marshaling
// depot and drum data structures.
type CreateFS interface {
	// Sub returns a filesystem for a subdirectory.
	Sub(name string) (sub CreateFS, err error)
	// Create creates a new file for writing.
	Create(name string) (file io.WriteCloser, err error)
	// Mkdir creates a new directory with the specified permissions.
	Mkdir(name string, filemode fs.FileMode) (err error)
}
