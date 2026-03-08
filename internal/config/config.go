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

var validLabel = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

const (
	LogModeFull       = "full"
	LogModeMinimal    = "minimal"
	LogModeOff        = "off"
	DefaultBaseDomain = "test"
)

type Route struct {
	Path string `yaml:"path"`
	Port int    `yaml:"port"`
}

type Domain struct {
	Name string `yaml:"name"`
	// Hostname stores the base domain suffix (for example, "local.example.com").
	// The field name is kept for on-disk compatibility with existing config files.
	Hostname string  `yaml:"hostname,omitempty"`
	Port     int     `yaml:"port"`
	Routes   []Route `yaml:"routes,omitempty"`
}

type Config struct {
	Domains []Domain `yaml:"domains"`
	LogMode string   `yaml:"log_mode,omitempty"`
	Cors    bool     `yaml:"cors,omitempty"`
}

func ValidateRoute(path string, port int) error {
	if path == "" || path[0] != '/' {
		return fmt.Errorf("route path must start with /")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid route port %d: must be between 1 and 65535", port)
	}
	return nil
}

func (d *Domain) MatchRoute(reqPath string) int {
	bestLen := 0
	bestPort := d.Port
	for _, r := range d.Routes {
		if len(r.Path) <= bestLen {
			continue
		}
		if reqPath == r.Path || (strings.HasPrefix(reqPath, r.Path) && (r.Path[len(r.Path)-1] == '/' || (len(reqPath) > len(r.Path) && reqPath[len(r.Path)] == '/'))) {
			bestLen = len(r.Path)
			bestPort = r.Port
		}
	}
	return bestPort
}

func DefaultHostname(name string) string {
	return ResolveHostname(name, "")
}

// NormalizeHostname lowercases, trims whitespace, and removes leading/trailing dots.
func NormalizeHostname(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	return strings.Trim(h, ".")
}

func ResolveHostname(name string, baseDomain string) string {
	baseDomain = NormalizeHostname(baseDomain)
	if baseDomain == "" {
		baseDomain = DefaultBaseDomain
	}
	return name + "." + baseDomain
}

func (d Domain) ResolvedHostname() string {
	return ResolveHostname(d.Name, d.Hostname)
}

func ValidateDomainName(name string) error {
	if name == "" {
		return fmt.Errorf("domain name cannot be empty")
	}
	if len(name) > 253 {
		return fmt.Errorf("domain name %q is too long: must be 253 characters or fewer", name)
	}
	for _, label := range strings.Split(name, ".") {
		if len(label) > 63 {
			return fmt.Errorf("domain label %q is too long: must be 63 characters or fewer", label)
		}
		if !validLabel.MatchString(label) {
			return fmt.Errorf("invalid domain name %q: labels must be lowercase alphanumeric with hyphens", name)
		}
	}
	return nil
}

func ValidateDomain(name string, port int) error {
	if err := ValidateDomainName(name); err != nil {
		return err
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

// FindDomain looks up a domain by Name first, then by ResolvedHostname.
// Name match takes priority because it appears first in the check.
func (c *Config) FindDomain(name string) (*Domain, int) {
	name = NormalizeHostname(name)
	for i := range c.Domains {
		if c.Domains[i].Name == name || c.Domains[i].ResolvedHostname() == name {
			return &c.Domains[i], i
		}
	}
	return nil, -1
}

func (c *Config) SetDomain(name string, port int, routes []Route) error {
	return c.SetDomainHostname(name, "", port, routes)
}

// SetDomainFields updates or appends a domain in memory without saving.
// Use this for batch updates, then call Save() once at the end.
func (c *Config) SetDomainFields(name string, hostname string, port int, routes []Route) {
	hostname = NormalizeHostname(hostname)
	if existing, idx := c.FindDomain(name); existing != nil {
		c.Domains[idx].Name = name
		c.Domains[idx].Hostname = hostname
		c.Domains[idx].Port = port
		c.Domains[idx].Routes = routes
		return
	}
	c.Domains = append(c.Domains, Domain{Name: name, Hostname: hostname, Port: port, Routes: routes})
}

func (c *Config) SetDomainHostname(name string, hostname string, port int, routes []Route) error {
	hostname = NormalizeHostname(hostname)
	if existing, idx := c.FindDomain(name); existing != nil {
		c.Domains[idx].Name = name
		c.Domains[idx].Hostname = hostname
		c.Domains[idx].Port = port
		c.Domains[idx].Routes = routes
		return c.Save()
	}
	c.Domains = append(c.Domains, Domain{Name: name, Hostname: hostname, Port: port, Routes: routes})
	return c.Save()
}

func (c *Config) RemoveDomain(name string) error {
	_, idx := c.FindDomain(name)
	if idx == -1 {
		return fmt.Errorf("domain %s not found", name)
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
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()

	return fn()
}
