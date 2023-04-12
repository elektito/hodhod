package hodhod

import (
	"fmt"
	"io"
)

type ErrorResponse struct {
	StatusCode         int
	Meta               string
	returnedStatusLine bool
}

func (resp *ErrorResponse) Backend() string {
	return "error"
}

func (resp *ErrorResponse) Init(req *Request) (err error) {
	return
}

func (resp *ErrorResponse) Read(p []byte) (n int, err error) {
	if resp.returnedStatusLine {
		return 0, io.EOF
	}

	status := []byte(fmt.Sprintf("%d %s\r\n", resp.StatusCode, resp.Meta))
	if len(p) < len(status) {
		return 0, fmt.Errorf("Not enough space in read buffer")
	}
	copy(p, status)

	n = len(status)
	resp.returnedStatusLine = true
	return
}

func (resp *ErrorResponse) Close() {

}

var _ Response = (*ErrorResponse)(nil)
