package hodhod

import (
	"fmt"
	"io"
)

type RedirectResponse struct {
	StatusCode         int
	Target             string
	returnedStatusLine bool
}

func (resp *RedirectResponse) Backend() string {
	return "redirect"
}

func (resp *RedirectResponse) Init(req *Request) (err error) {
	return
}

func (resp *RedirectResponse) Read(p []byte) (n int, err error) {
	if resp.returnedStatusLine {
		return 0, io.EOF
	}

	status := []byte(fmt.Sprintf("%d %s\r\n", resp.StatusCode, resp.Target))
	if len(p) < len(status) {
		return 0, fmt.Errorf("Not enough space in read buffer")
	}
	copy(p, status)

	n = len(status)
	resp.returnedStatusLine = true
	return
}

func (resp *RedirectResponse) Close() {
}

func NewTempRedirectResp(target string) (resp Response) {
	return &RedirectResponse{
		StatusCode: 30,
		Target:     target,
	}
}

func NewPermRedirectResp(target string) (resp Response) {
	return &RedirectResponse{
		StatusCode: 31,
		Target:     target,
	}
}

var _ Response = (*RedirectResponse)(nil)
