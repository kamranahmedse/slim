package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kamranahmedse/slim/internal/config"
	"gopkg.in/yaml.v3"
)

const FileName = ".slim.yaml"

var (
	getwdFn = os.Getwd
	statFn  = os.Stat
)

type Service struct {
	Domain string         `yaml:"domain"`
	Port   int            `yaml:"port"`
	Routes []config.Route `yaml:"routes,omitempty"`
}

type ProjectConfig struct {
	Services []Service `yaml:"services"`
	LogMode  string    `yaml:"log_mode,omitempty"`
}

func Find() (string, error) {
	dir, err := getwdFn()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	for {
		path := filepath.Join(dir, FileName)
		if _, err := statFn(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found (searched up to filesystem root)", FileName)
		}
		dir = parent
	}
}

func Load(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var pc ProjectConfig
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &pc, nil
}

func Discover() (*ProjectConfig, string, error) {
	path, err := Find()
	if err != nil {
		return nil, "", err
	}

	pc, err := Load(path)
	if err != nil {
		return nil, "", err
	}

	return pc, path, nil
}

func (pc *ProjectConfig) Validate() error {
	if len(pc.Services) == 0 {
		return fmt.Errorf("no services defined in %s", FileName)
	}

	if pc.LogMode != "" {
		if err := config.ValidateLogMode(pc.LogMode); err != nil {
			return err
		}
	}

	seen := make(map[string]bool)
	for _, svc := range pc.Services {
		if err := config.ValidateDomain(svc.Domain, svc.Port); err != nil {
			return fmt.Errorf("service %q: %w", svc.Domain, err)
		}
		if seen[svc.Domain] {
			return fmt.Errorf("duplicate domain %q", svc.Domain)
		}
		seen[svc.Domain] = true

		for _, r := range svc.Routes {
			if err := config.ValidateRoute(r.Path, r.Port); err != nil {
				return fmt.Errorf("service %q route %q: %w", svc.Domain, r.Path, err)
			}
		}
	}

	return nil
}
