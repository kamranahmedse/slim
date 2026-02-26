package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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

	if err := os.WriteFile(Path(), data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

func (c *Config) FindDomain(name string) (*Domain, int) {
	for i := range c.Domains {
		if c.Domains[i].Name == name {
			return &c.Domains[i], i
		}
	}
	return nil, -1
}

func LogPath() string {
	return filepath.Join(Dir(), "access.log")
}

func (c *Config) AddDomain(name string, port int) error {
	if existing, _ := c.FindDomain(name); existing != nil {
		return fmt.Errorf("domain %s.local already exists (port %d)", name, existing.Port)
	}
	c.Domains = append(c.Domains, Domain{Name: name, Port: port})
	return c.Save()
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
