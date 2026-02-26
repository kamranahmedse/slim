//go:build linux

package system

import "errors"

type linuxPortFwd struct{}

func NewPortForwarder() PortForwarder {
	return &linuxPortFwd{}
}

func (l *linuxPortFwd) Enable() error {
	return errors.New("port forwarding not yet implemented on Linux")
}

func (l *linuxPortFwd) Disable() error {
	return errors.New("port forwarding not yet implemented on Linux")
}

func (l *linuxPortFwd) IsEnabled() bool {
	return false
}
