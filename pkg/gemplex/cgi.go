package gemplex

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"
)

type CgiResponse struct {
	cmd          *exec.Cmd
	stdout       io.Reader
	cancelScript func()
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

func (resp CgiResponse) Close() {
	if resp.cmd.ProcessState == nil {
		resp.cancelScript()
	}
}

func NewCgiResp(req Request, scriptPath string, cfg *Config) (resp Response) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(cfg.CgiTimeout)*time.Second)
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
		resp = &ErrorResponse{
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
			w.CloseWithError(fmt.Errorf("CGI timeout"))
		} else {
			w.Close()
		}
	}()

	resp = CgiResponse{
		cmd:          cmd,
		stdout:       r,
		cancelScript: cancelFunc,
	}
	return
}

var _ Response = (*CgiResponse)(nil)
var _ error = (*CgiError)(nil)
