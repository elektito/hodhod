package hodhod

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
	stdin        io.WriteCloser
	stdout       io.Reader
	stderr       io.Reader
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

func (resp *CgiResponse) Backend() string {
	return "cgi"
}

func (resp *CgiResponse) Init(req *Request) (err error) {
	reqLine := []byte(req.Url.String())
	reqLine = append(reqLine, '\r', '\n')
	_, err = resp.stdin.Write(reqLine)
	resp.stdin.Close()
	return
}

func (resp *CgiResponse) Read(p []byte) (n int, err error) {
	if resp.cmd.ProcessState != nil {
		if resp.cmd.ProcessState.ExitCode() != 0 {
			err = cgiError(resp.cmd.ProcessState.ExitCode())
			return
		}

		err = io.EOF
		return
	}

	n, err = resp.stdout.Read(p)
	return
}

func (resp *CgiResponse) Close() {
	if resp.cmd.ProcessState == nil {
		resp.cancelScript()
	}
}

func NewCgiResp(req Request, scriptPath string, cfg *Config) (resp Response) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(cfg.CgiTimeout)*time.Second)
	cmd := exec.CommandContext(ctx, scriptPath)

	rStdin, wStdin := io.Pipe()
	rStdout, wStdout := io.Pipe()
	rStderr, wStderr := io.Pipe()

	cmd.Env = []string{
		"GATEWAY_INTERFACE=CGI/1.1",
		"SERVER_PROTOCOL=GEMINI",
		"REQUEST_METHOD=",
		"SERVER_SOFTWARE=hodhod",
		fmt.Sprintf("GEMINI_URL=%s", req.Url.String()),
		fmt.Sprintf("GEMINI_URL_PATH=%s", req.Url.Path),
		fmt.Sprintf("PATH_INFO=%s", req.Url.Path),
		fmt.Sprintf("QUERY_STRING=%s", req.Url.RawQuery),
		fmt.Sprintf("SCRIPT_NAME=%s", scriptPath),
		fmt.Sprintf("SERVER_NAME=%s", req.Url.Hostname()),
		fmt.Sprintf("REMOTE_ADDR=%s", req.RemoteAddr),
		fmt.Sprintf("REMOTE_HOST=%s", req.RemoteAddr),
	}
	cmd.Stdin = rStdin
	cmd.Stdout = wStdout
	cmd.Stderr = wStderr
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
		// This function can be useful for debugging CGI scripts. We can read
		// the stderr here and log it.
		//
		// TODO: We could have an option to log these (maybe to a separate file,
		// and/or when there was a CGI error)
		//
		// stderr, err := io.ReadAll(rStderr)
		// if err == nil {
		//    log.Println("CGI stderr:", string(stderr))
		// } else {
		// 	  log.Println("Error reading CGI stderr:", err)
		// }

		io.Copy(io.Discard, rStderr)
	}()

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("CGI script (%s) timeout (error: %s)\n", scriptPath, err)
			rStdin.CloseWithError(fmt.Errorf("CGI timeout"))
			wStdout.CloseWithError(fmt.Errorf("CGI timeout"))
			wStderr.CloseWithError(fmt.Errorf("CGI timeout"))
		} else {
			rStdin.Close()
			wStdout.Close()
			wStderr.Close()
		}
	}()

	resp = &CgiResponse{
		cmd:          cmd,
		stdin:        wStdin,
		stdout:       rStdout,
		stderr:       rStderr,
		cancelScript: cancelFunc,
	}
	return
}

var _ Response = (*CgiResponse)(nil)
var _ error = (*CgiError)(nil)
