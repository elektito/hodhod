package response

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/elektito/gemplex/pkg/config"
)

type Response interface {
	Read(p []byte) (n int, err error)
	WriteStatus(w io.Writer) (err error)
}

type StaticResponse struct {
	file       *os.File
	statusCode int
	meta       string
}

type CgiResponse struct {
	cmd *exec.Cmd
}

func (resp StaticResponse) Read(p []byte) (n int, err error) {
	return resp.file.Read(p)
}

func (resp StaticResponse) WriteStatus(w io.Writer) (err error) {
	status := fmt.Sprintf("%d %s\r\n", resp.statusCode, resp.meta)
	_, err = w.Write([]byte(status))
	return
}

func NewFileResp(filename string, cfg *config.GemplexConfig) (resp Response) {
	f, err := os.Open(filename)

	if err == nil {
		info, serr := f.Stat()
		if serr == nil && info.IsDir() {
			filename = path.Join(filename, cfg.MatchOptions.IndexFilename)
			f, err = os.Open(filename)
		}
	}

	if err != nil {
		for _, ext := range cfg.MatchOptions.DefaultExts {
			f, err = os.Open(filename + "." + ext)
			if err == nil {
				break
			}
		}
	}

	if err != nil {
		resp = StaticResponse{
			file:       f,
			statusCode: 51,
			meta:       "Not Found",
		}
		return
	}

	resp = StaticResponse{
		file:       f,
		statusCode: 20,
		meta:       "text/gemini", // TODO find a way of detecting content type
	}
	return
}

var _ Response = (*StaticResponse)(nil)
var _ io.Reader = Response(nil)
