//go:build !darwin && !linux

package cert

import "errors"

func TrustCA() error {
	return errors.New("trusting CA is only supported on macOS and Linux")
}

func UntrustCA() error {
	return errors.New("untrusting CA is only supported on macOS and Linux")
}
