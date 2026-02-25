//go:build linux

package portfwd

import "fmt"

type LinuxPortFwd struct{}

func New() PortForwarder {
	return &LinuxPortFwd{}
}

func (l *LinuxPortFwd) Enable() error {
	return fmt.Errorf("port forwarding not yet implemented on Linux")
}

func (l *LinuxPortFwd) Disable() error {
	return fmt.Errorf("port forwarding not yet implemented on Linux")
}

func (l *LinuxPortFwd) IsEnabled() bool {
	return false
}
