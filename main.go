package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/elektito/gemplex/pkg/config"
)

const (
	ConnectionTimeout = 30 * time.Second
)

func fail(whileDoing string, err error) {
	log.Printf("Error %s: %s\n", whileDoing, err)
	os.Exit(1)
}

func getUpstreamFromClientHello(hello *tls.ClientHelloInfo, cfg *config.GemplexConfig) (conn net.Conn, err error) {
	upstream := cfg.GetUpstreamByHostname(hello.ServerName)
	if upstream == nil {
		err = fmt.Errorf("No route found for server name: %s", hello.ServerName)
		return
	}

	conn, err = net.Dial("tcp", upstream.Addr)
	return
}

// code shamelessly copied from:
// https://www.agwa.name/blog/post/writing_an_sni_proxy_in_go
type readOnlyConn struct {
	reader io.Reader
}

func (conn readOnlyConn) Read(p []byte) (int, error)         { return conn.reader.Read(p) }
func (conn readOnlyConn) Write(p []byte) (int, error)        { return 0, io.ErrClosedPipe }
func (conn readOnlyConn) Close() error                       { return nil }
func (conn readOnlyConn) LocalAddr() net.Addr                { return nil }
func (conn readOnlyConn) RemoteAddr() net.Addr               { return nil }
func (conn readOnlyConn) SetDeadline(t time.Time) error      { return nil }
func (conn readOnlyConn) SetReadDeadline(t time.Time) error  { return nil }
func (conn readOnlyConn) SetWriteDeadline(t time.Time) error { return nil }

// code shamelessly copied from:
// https://www.agwa.name/blog/post/writing_an_sni_proxy_in_go
func readClientHello(reader io.Reader) (*tls.ClientHelloInfo, error) {
	var hello *tls.ClientHelloInfo

	err := tls.Server(readOnlyConn{reader: reader}, &tls.Config{
		GetConfigForClient: func(argHello *tls.ClientHelloInfo) (*tls.Config, error) {
			hello = new(tls.ClientHelloInfo)
			*hello = *argHello
			return nil, nil
		},
	}).Handshake()

	if hello == nil {
		return nil, err
	}

	return hello, nil
}

// code shamelessly copied from:
// https://www.agwa.name/blog/post/writing_an_sni_proxy_in_go
func peekClientHello(reader io.Reader) (hello *tls.ClientHelloInfo, outReader io.Reader, err error) {
	peekedBytes := new(bytes.Buffer)
	hello, err = readClientHello(io.TeeReader(reader, peekedBytes))
	if err != nil {
		return
	}

	//outReader = io.MultiReader(peekedBytes, reader)
	outReader = peekedBytes
	return
}

// code shamelessly copied from:
// https://www.agwa.name/blog/post/writing_an_sni_proxy_in_go
func handleConn(clientConn net.Conn, cfg *config.GemplexConfig) {
	defer clientConn.Close()

	log.Println("Accepted connection.")

	err := clientConn.SetDeadline(time.Now().Add(ConnectionTimeout))
	if err != nil {
		log.Println("Error setting connection deadline:", err)
		return
	}

	clientHello, clientHelloBytes, err := peekClientHello(clientConn)
	if err != nil {
		log.Println("Error reading ClientHello:", err)
		return
	}

	upstreamConn, err := getUpstreamFromClientHello(clientHello, cfg)
	if err != nil {
		log.Println("Error getting upstream from ClientHello:", err)
		return
	}
	defer upstreamConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		io.Copy(clientConn, upstreamConn)
		clientConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()

	go func() {
		io.Copy(upstreamConn, clientHelloBytes)
		io.Copy(upstreamConn, clientConn)
		upstreamConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()

	wg.Wait()

	log.Println("Closed connection.")
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fail("loading config", err)
	}

	listener, err := net.Listen("tcp", cfg.ListenAddr)
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
