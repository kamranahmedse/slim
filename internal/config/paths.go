package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	ProxyHTTPPort  = 10080
	ProxyHTTPSPort = 10443

	TunnelServerURL = "wss://app.slim.sh/tunnel"
)

var baseDir string

func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	baseDir = filepath.Join(home, ".slim")
	return nil
}

func Dir() string {
	return baseDir
}

func Path() string {
	return filepath.Join(Dir(), "config.yaml")
}

func LogPath() string {
	return filepath.Join(Dir(), "access.log")
}

func SocketPath() string {
	return filepath.Join(Dir(), "slim.sock")
}

func PidPath() string {
	return filepath.Join(Dir(), "slim.pid")
}

func TunnelTokenPath() string {
	return filepath.Join(Dir(), "tunnel-token")
}

func AuthPath() string {
	return filepath.Join(Dir(), "auth.json")
}
