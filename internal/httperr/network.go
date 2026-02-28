package httperr

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

func NetworkHint(err error) string {
	if err == nil {
		return ""
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "could not resolve host — check your internet connection"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "connection timed out — check your internet connection"
	}

	msg := err.Error()
	if strings.Contains(msg, "connection refused") {
		return "connection refused — the server may be down"
	}
	if strings.Contains(msg, "no such host") {
		return "could not resolve host — check your internet connection"
	}
	if strings.Contains(msg, "network is unreachable") || strings.Contains(msg, "no route to host") {
		return "network is unreachable — check your internet connection"
	}

	return msg
}

func Wrap(context string, err error) error {
	if err == nil {
		return nil
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return fmt.Errorf("%s: %s", context, NetworkHint(err))
	}

	return fmt.Errorf("%s: %w", context, err)
}
