package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	ProxyHTTPPort  = 10080
	ProxyHTTPSPort = 10443

	defaultAPIBase      = "https://app.slim.sh"
	defaultTunnelServer = "wss://app.slim.sh/tunnel"
)

func APIBaseURL() string {
	if v := os.Getenv("SLIM_TUNNEL_SERVER_API"); v != "" {
		return v
	}
	return defaultAPIBase
}

func TunnelServerURL() string {
	if v := os.Getenv("SLIM_TUNNEL_SERVER"); v != "" {
		return v
	}
	return defaultTunnelServer
}

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
