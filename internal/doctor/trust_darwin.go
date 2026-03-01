//go:build darwin

package doctor

import (
	"os/exec"

	"github.com/kamranahmedse/slim/internal/cert"
)

var execCommandFn = exec.Command

func verifyCAIsTrusted() CheckResult {
	name := "CA trust"

	cmd := execCommandFn("security", "verify-cert", "-c", cert.CACertPath())
	if err := cmd.Run(); err != nil {
		return CheckResult{Name: name, Status: Fail, Message: "not trusted by OS"}
	}

	return CheckResult{Name: name, Status: Pass, Message: "trusted by OS"}
}
