package hodhod

import (
	"io"
)

type Response interface {
	// Read a part of the response body. Implementes the io.Reader interface.
	Read(p []byte) (n int, err error)

	// Release any resources related to this response
	Close()
}

var _ io.Reader = Response(nil)
