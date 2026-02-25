package portfwd

type PortForwarder interface {
	Enable() error
	Disable() error
	IsEnabled() bool
}
