package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"github.com/elektito/gemplex/pkg/config"
	"github.com/elektito/gemplex/pkg/response"
)

const (
	ConnectionTimeout = 30 * time.Second

	// This is the amount specified by the Gemini spec
	GeminiMaxRequestSize = 1024
)

var ErrNotFound = errors.New("No route found for url")

func fail(whileDoing string, err error) {
	log.Printf("Error %s: %s\n", whileDoing, err)
	os.Exit(1)
}

func getResponseForRequest(req string, cfg *config.GemplexConfig) (resp response.Response, err error) {
	backend, unmatched := cfg.GetBackendByUrl(req)
	if backend == nil {
		err = ErrNotFound
		return
	}

	if backend.Type == "static" {
		filename := path.Join(backend.Location, unmatched)
		return response.NewFileResp(filename)
	}

	return
}

func handleConn(conn net.Conn, cfg *config.GemplexConfig) {
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

	req := s.Text()
	resp, err := getResponseForRequest(req, cfg)
	if err != nil {
		log.Println("Could not find response for the request:", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		resp.WriteStatus(conn)
		io.Copy(conn, resp)
		conn.(*tls.Conn).NetConn().(*net.TCPConn).CloseWrite()
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
			log.Println("Error reading from client:", err)
			conn.Close()
		}

		wg.Done()
	}()

	wg.Wait()

	log.Println("Closed connection.")
}

func loadCertificates(cfg *config.GemplexConfig) (certs []tls.Certificate, err error) {
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
	cfg, err := config.Load()
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
