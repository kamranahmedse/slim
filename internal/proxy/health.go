package proxy

import (
	"fmt"
	"net"
	"time"
)

func CheckUpstream(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
