package proxy

import (
	"fmt"
	"net"
	"time"
)

const upstreamPollInterval = 200 * time.Millisecond

func CheckUpstream(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func WaitForUpstream(port int, timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}

	if CheckUpstream(port) {
		return nil
	}

	ticker := time.NewTicker(upstreamPollInterval)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if CheckUpstream(port) {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("upstream localhost:%d did not become reachable within %s", port, timeout)
		}
	}
}
