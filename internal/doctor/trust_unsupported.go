//go:build !darwin && !linux

package doctor

func verifyCAIsTrusted() CheckResult {
	return CheckResult{
		Name:    "CA trust",
		Status:  Warn,
		Message: "trust verification not supported on this platform",
	}
}
