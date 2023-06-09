package hodhod

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

type StaticResponse struct {
	file               *os.File
	contentType        string
	returnedStatusLine bool
}

func (resp *StaticResponse) Backend() string {
	return "static"
}

func (resp *StaticResponse) Init(req *Request) (err error) {
	return
}

func (resp *StaticResponse) Read(p []byte) (n int, err error) {
	if !resp.returnedStatusLine {
		status := []byte(fmt.Sprintf("20 %s\r\n", resp.contentType))
		if len(p) < len(status) {
			return 0, fmt.Errorf("Not enough space in read buffer")
		}
		copy(p, status)
		resp.returnedStatusLine = true
		n = len(status)
		return
	}

	return resp.file.Read(p)
}

func (resp *StaticResponse) Close() {
	if resp.file != nil {
		resp.file.Close()
	}
}

func NewFileResp(filename string, req Request, cfg *Config) (resp Response) {
	isDir := false
	f, err := os.Open(filename)

	if err == nil {
		info, serr := f.Stat()
		if serr == nil && info.IsDir() {
			isDir = true
			filename = path.Join(filename, cfg.MatchOptions.IndexFilename)
			f, err = os.Open(filename)
		}
	}

	if err != nil {
		for _, ext := range cfg.MatchOptions.DefaultExts {
			f, err = os.Open(filename + "." + ext)
			if err == nil {
				filename = filename + "." + ext
				break
			}
		}
	}

	if err != nil {
		resp = &ErrorResponse{
			StatusCode: 51,
			Meta:       "Not Found",
		}
		return
	}

	u := *req.Url
	if isDir && u.Path[len(u.Path)-1] != '/' {
		u.Path = u.Path + "/"
		return NewPermRedirectResp(u.String())
	} else if !isDir && u.Path[len(u.Path)-1] == '/' {
		u.Path = u.Path[:len(u.Path)-1]
		return NewPermRedirectResp(u.String())
	}

	ext := filepath.Ext(filename)
	if ext != "" {
		// remove leading dot
		ext = ext[1:]
	}
	contentType, ok := cfg.ContentType.ExtMap[ext]
	if !ok {
		contentType = cfg.ContentType.Default
	}

	resp = &StaticResponse{
		file:        f,
		contentType: contentType,
	}
	return
}

var _ Response = (*StaticResponse)(nil)
