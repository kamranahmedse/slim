package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/kamranahmedse/localname/internal/config"
)

type IPCServer struct {
	listener net.Listener
	handler  func(Request) Response
}

func NewIPCServer(handler func(Request) Response) (*IPCServer, error) {
	sockPath := config.SocketPath()

	os.Remove(sockPath)
	if err := os.MkdirAll(filepath.Dir(sockPath), 0755); err != nil {
		return nil, err
	}

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("listening on socket: %w", err)
	}

	return &IPCServer{listener: ln, handler: handler}, nil
}

func (s *IPCServer) Serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *IPCServer) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		resp := Response{OK: false, Error: err.Error()}
		json.NewEncoder(conn).Encode(resp)
		return
	}

	resp := s.handler(req)
	json.NewEncoder(conn).Encode(resp)
}

func (s *IPCServer) Close() {
	s.listener.Close()
	os.Remove(config.SocketPath())
}

func SendIPC(req Request) (*Response, error) {
	conn, err := net.DialTimeout("unix", config.SocketPath(), 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connecting to daemon: %w (is localname running?)", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return &resp, nil
}
