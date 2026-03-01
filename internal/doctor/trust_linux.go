//go:build linux

package doctor

import (
	"os"
	"path/filepath"

	"github.com/kamranahmedse/slim/internal/cert"
)

var caDirs = []string{
	"/usr/local/share/ca-certificates",
	"/etc/pki/ca-trust/source/anchors",
	"/etc/ca-certificates/trust-source/anchors",
}

func verifyCAIsTrusted() CheckResult {
	name := "CA trust"
	caBase := filepath.Base(cert.CACertPath())

	for _, dir := range caDirs {
		anchor := filepath.Join(dir, caBase)
		if _, err := os.Stat(anchor); err == nil {
			return CheckResult{Name: name, Status: Pass, Message: "trusted by OS (found in " + dir + ")"}
		}
	}

	return CheckResult{Name: name, Status: Fail, Message: "not found in system CA directories"}
}
