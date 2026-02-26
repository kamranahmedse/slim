package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

var validName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

type Domain struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type Config struct {
	Domains []Domain `yaml:"domains"`
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".localname")
}

func Path() string {
	return filepath.Join(Dir(), "config.yaml")
}

func LogPath() string {
	return filepath.Join(Dir(), "access.log")
}

func SocketPath() string {
	return filepath.Join(Dir(), "localname.sock")
}

func PidPath() string {
	return filepath.Join(Dir(), "localname.pid")
}

func ValidateDomain(name string, port int) error {
	if name == "" {
		return fmt.Errorf("domain name cannot be empty")
	}
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid domain name %q: must be lowercase alphanumeric with hyphens", name)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
	}
	return nil
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

func (c *Config) AddDomain(name string, port int) error {
	if err := ValidateDomain(name, port); err != nil {
		return err
	}
	if existing, _ := c.FindDomain(name); existing != nil {
		return fmt.Errorf("domain %s.local already exists (port %d)", name, existing.Port)
	}
	c.Domains = append(c.Domains, Domain{Name: name, Port: port})
	return c.Save()
}

func (c *Config) SetDomain(name string, port int) error {
	if err := ValidateDomain(name, port); err != nil {
		return err
	}
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
