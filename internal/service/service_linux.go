//go:build linux

package service

import "fmt"

type LinuxService struct{}

func New() Service {
	return &LinuxService{}
}

func (l *LinuxService) Install(binaryPath string) error {
	return fmt.Errorf("service install not yet implemented on Linux")
}

func (l *LinuxService) Uninstall() error {
	return fmt.Errorf("service uninstall not yet implemented on Linux")
}

func (l *LinuxService) IsInstalled() bool {
	return false
}
