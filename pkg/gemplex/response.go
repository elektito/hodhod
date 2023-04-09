package gemplex

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"time"
)

var CgiTimeout = 10 * time.Second

type Response interface {
	// Read a part of the response body. Implementes the io.Reader interface.
	Read(p []byte) (n int, err error)

	// Release any resources related to this response
	Close()

	// Write the gemini status line (including the CRLF) to the given io.Writer
	WriteStatus(w io.Writer) (err error)
}

type StaticResponse struct {
	file        *os.File
	contentType string
}

type CgiResponse struct {
	cmd          *exec.Cmd
	stdout       io.Reader
	cancelScript func()
}

type ErrorResponse struct {
	StatusCode int
	Meta       string
}

type CgiError struct {
	ExitCode int
}

func (e CgiError) Error() string {
	if e.ExitCode != 0 {
		return fmt.Sprintf("CGI script exited with non-zero exit code %d.", e.ExitCode)
	}

	return "Error running CGI script"
}

func cgiError(exitCode int) CgiError {
	return CgiError{
		ExitCode: exitCode,
	}
}

func (resp StaticResponse) Read(p []byte) (n int, err error) {
	return resp.file.Read(p)
}

func (resp StaticResponse) WriteStatus(w io.Writer) (err error) {
	status := fmt.Sprintf("20 %s\r\n", resp.contentType)
	_, err = w.Write([]byte(status))
	return
}

func (resp StaticResponse) Close() {
	if resp.file != nil {
		resp.file.Close()
	}
}

func (resp CgiResponse) Read(p []byte) (n int, err error) {
	if resp.cmd.ProcessState != nil {
		if resp.cmd.ProcessState.ExitCode() != 0 {
			err = cgiError(resp.cmd.ProcessState.ExitCode())
			return
		}

		err = io.EOF
		return
	}

	return resp.stdout.Read(p)
}

func (resp CgiResponse) WriteStatus(w io.Writer) (err error) {
	// the cgi script writes the status line, so we won't write anything here.
	return
}

func (resp CgiResponse) Close() {
	if resp.cmd.ProcessState == nil {
		resp.cancelScript()
	}
}

func (resp ErrorResponse) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (resp ErrorResponse) WriteStatus(w io.Writer) (err error) {
	s := fmt.Sprintf("%d %s\r\n", resp.StatusCode, resp.Meta)
	w.Write([]byte(s))
	return nil
}

func (resp ErrorResponse) Close() {

}

func NewFileResp(filename string, cfg *Config) (resp Response) {
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
		resp = ErrorResponse{
			StatusCode: 51,
			Meta:       "Not Found",
		}
		return
	}

	resp = StaticResponse{
		file:        f,
		contentType: "text/gemini", // TODO find a way of detecting content type
	}
	return
}

func NewCgiResp(req Request, scriptPath string, cfg *Config) (resp Response) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), CgiTimeout)
	cmd := exec.CommandContext(ctx, scriptPath)

	// create a pipe to connect to the script's stdout; we set the writer as the
	// script's stdout writer (where its output is written to), and keep the
	// reader side so we can read from and send the response to the client.
	r, w := io.Pipe()

	cmd.Env = []string{
		"GATEWAY_INTERFACE=CGI/1.1",
		"SERVER_PROTOCOL=GEMINI",
		"REQUEST_METHOD=",
		"SERVER_SOFTWARE=gemplex",
		fmt.Sprintf("GEMINI_URL=%s", req.Url.String()),
		fmt.Sprintf("GEMINI_URL_PATH=%s", req.Url.Path),
		fmt.Sprintf("PATH_INFO=%s", req.Url.Path),
		fmt.Sprintf("QUERY_STRING=%s", req.Url.RawQuery),
		fmt.Sprintf("SCRIPT_NAME=%s", scriptPath),
		fmt.Sprintf("SERVER_NAME=%s", req.Url.Hostname()),
		fmt.Sprintf("REMOTE_ADDR=%s", req.RemoteAddr),
		fmt.Sprintf("REMOTE_HOST=%s", req.RemoteAddr),
	}
	cmd.Stdout = w
	cmd.WaitDelay = 5 * time.Second

	err := cmd.Start()
	if err != nil {
		log.Println("Error running CGI script:", err)
		resp = ErrorResponse{
			StatusCode: 43,
			Meta:       "CGI Error",
		}

		cancelFunc()
		return
	}

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Println("CGI script timeout:", scriptPath)
			r.CloseWithError(fmt.Errorf("CGI timeout"))
		} else {
			r.Close()
		}
	}()

	resp = CgiResponse{
		cmd:          cmd,
		stdout:       r,
		cancelScript: cancelFunc,
	}
	return
}

var _ Response = (*StaticResponse)(nil)
var _ Response = (*CgiResponse)(nil)
var _ Response = (*ErrorResponse)(nil)
var _ io.Reader = Response(nil)

var _ error = (*CgiError)(nil)
