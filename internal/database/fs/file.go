package fs

import "io"

type File interface {
	io.Writer
	io.Closer
}
