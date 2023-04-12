package hodhod

import (
	"io"
)

type Response interface {
	// Called before response body is read, in order to perform any needed
	// initialization.
	Init(req *Request) (err error)

	// Read a part of the response body. Implementes the io.Reader interface.
	Read(p []byte) (n int, err error)

	// Release any resources related to this response
	Close()

	// Returns the backend name for this response
	Backend() string
}

var _ io.Reader = Response(nil)
