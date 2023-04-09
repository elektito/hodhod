package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	"github.com/elektito/gemplex/pkg/gemplex"
)

const (
	ConnectionTimeout = 30 * time.Second

	// This is the amount specified by the Gemini spec
	GeminiMaxRequestSize = 1024
)

type ErrNotFound struct {
	Reason string
	Url    string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("URL %s not found: %s", e.Url, e.Reason)
}

var _ error = (*ErrNotFound)(nil)

func errNotFound(url string, reason string) ErrNotFound {
	return ErrNotFound{
		Url:    url,
		Reason: reason,
	}
}

func fail(whileDoing string, err error) {
	log.Printf("Error %s: %s\n", whileDoing, err)
	os.Exit(1)
}

func getResponseForRequest(req gemplex.Request, cfg *gemplex.Config) (resp gemplex.Response, err error) {
	backend, unmatched := cfg.GetBackendByUrl(req.Url)
	if backend == nil {
		err = errNotFound(req.Url.String(), "no route")
		return
	}

	if backend.Type == "static" {
		filename := path.Join(backend.Location, unmatched)
		resp = gemplex.NewFileResp(filename, cfg)
		return
	}

	if backend.Type == "cgi" {
		resp = gemplex.NewCgiResp(req, backend.Script, cfg)
		return
	}

	return
}

func handleConn(conn net.Conn, cfg *gemplex.Config) {
	defer conn.Close()

	log.Println("Accepted connection.")

	err := conn.SetDeadline(time.Now().Add(ConnectionTimeout))
	if err != nil {
		log.Println("Error setting connection deadline:", err)
		return
	}

	buf := make([]byte, GeminiMaxRequestSize)
	s := bufio.NewScanner(conn)
	s.Buffer(buf, GeminiMaxRequestSize)
	ok := s.Scan()
	if !ok {
		log.Println("Could not read request:", s.Err())
		return
	}

	urlStr := s.Text()
	urlParsed, err := url.Parse(urlStr)
	if err != nil {
		conn.Write([]byte("59 Bad Request\r\n"))
		return
	}
	req := gemplex.Request{
		Url:        urlParsed,
		RemoteAddr: conn.RemoteAddr().String(),
	}
	resp, err := getResponseForRequest(req, cfg)
	if err != nil {
		log.Println("Could not find response for the request:", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer resp.Close()
		_, err := io.Copy(conn, resp)
		if err != nil {
			log.Println("Error sending response:", err)

			// close the underlying connection (instead of letting the tls
			// connection to be properly closed) to signal to the client that
			// there was an error.
			conn.(*tls.Conn).NetConn().Close()
		} else {
			conn.(*tls.Conn).NetConn().(*net.TCPConn).CloseWrite()
		}
		wg.Done()
	}()

	go func() {
		// the client should not send any more bytes; if we receive anything,
		// that's an error, and we'll close the connection.
		buf := make([]byte, 1)
		n, err := conn.Read(buf)
		if n != 0 {
			log.Println("Unexpected input from client.")
			conn.Close()
		} else if err != nil && err != io.EOF {
			conn.Close()
		}

		wg.Done()
	}()

	wg.Wait()

	log.Println("Closed connection.")
}

func loadCertificates(cfg *gemplex.Config) (certs []tls.Certificate, err error) {
	certs = make([]tls.Certificate, len(cfg.Certs))
	for i, c := range cfg.Certs {
		certs[i], err = tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return
		}

		// the documentation for the `Certificates` field of `tls.Config` says
		// that if the optional Leaf field is not set, and there are multiple
		// certificates, there will be a significant pre-handshake cost (because
		// the certificate needs to be parsed every time). Here, we parse the
		// leaf certificate and store it in the Leaf field so that this will not
		// happen.
		certs[i].Leaf, err = x509.ParseCertificate(certs[i].Certificate[0])
		if err != nil {
			return
		}
	}

	return
}

func main() {
	cfg, err := gemplex.LoadConfig()
	if err != nil {
		fail("loading config", err)
	}

	certs, err := loadCertificates(&cfg)
	if err != nil {
		fail("loading certificates", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: certs,
	}
	listener, err := tls.Listen("tcp", cfg.ListenAddr, tlsConfig)
	if err != nil {
		fail("starting listening", err)
	}

	log.Println("Started listening at:", cfg.ListenAddr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fail("accepting request", err)
		}

		go handleConn(conn, &cfg)
	}
}
