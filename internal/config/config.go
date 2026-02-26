package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"
)

var validName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

const (
	ProxyHTTPPort  = 10080
	ProxyHTTPSPort = 10443

	LogModeFull    = "full"
	LogModeMinimal = "minimal"
	LogModeOff     = "off"
)

type Domain struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type Config struct {
	Domains []Domain `yaml:"domains"`
	LogMode string   `yaml:"log_mode,omitempty"`
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

func ValidateDomain(name string, port int) error {
	if name == "" {
		return fmt.Errorf("domain name cannot be empty")
	}
	if len(name) > 63 {
		return fmt.Errorf("domain name %q is too long: must be 63 characters or fewer", name)
	}
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid domain name %q: must be lowercase alphanumeric with hyphens", name)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
	}
	return nil
}

func ValidateLogMode(mode string) error {
	switch normalizeLogMode(mode) {
	case LogModeFull, LogModeMinimal, LogModeOff:
		return nil
	default:
		return fmt.Errorf("invalid log mode %q: must be one of full|minimal|off", mode)
	}
}

func normalizeLogMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return LogModeFull
	}
	return mode
}

func (c *Config) EffectiveLogMode() string {
	return normalizeLogMode(c.LogMode)
}

func Load() (*Config, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(Dir(), 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(Path(), data, 0644)
}

func (c *Config) FindDomain(name string) (*Domain, int) {
	for i := range c.Domains {
		if c.Domains[i].Name == name {
			return &c.Domains[i], i
		}
	}
	return nil, -1
}

func (c *Config) SetDomain(name string, port int) error {
	if existing, idx := c.FindDomain(name); existing != nil {
		c.Domains[idx].Port = port
		return c.Save()
	}
	c.Domains = append(c.Domains, Domain{Name: name, Port: port})
	return c.Save()
}

func (c *Config) RemoveDomain(name string) error {
	_, idx := c.FindDomain(name)
	if idx == -1 {
		return fmt.Errorf("domain %s.local not found", name)
	}
	c.Domains = append(c.Domains[:idx], c.Domains[idx+1:]...)
	return c.Save()
}

func WithLock(fn func() error) error {
	if err := os.MkdirAll(Dir(), 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	lockPath := filepath.Join(Dir(), "config.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening lock file: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquiring config lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	return fn()
}
