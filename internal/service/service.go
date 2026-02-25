package service

type Service interface {
	Install(binaryPath string) error
	Uninstall() error
	IsInstalled() bool
}
