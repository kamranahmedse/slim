package doctor

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/kamranahmedse/slim/internal/cert"
	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/system"
)

type Status int

const (
	Pass Status = iota
	Warn
	Fail
)

type CheckResult struct {
	Name    string
	Status  Status
	Message string
}

type Report struct {
	Results []CheckResult
}

var (
	readFileFn        = os.ReadFile
	daemonIsRunningFn = daemon.IsRunning
	daemonSendIPCFn   = daemon.SendIPC
	newPortFwdFn      = system.NewPortForwarder
	configLoadFn      = config.Load
	dialTimeoutFn     = net.DialTimeout
)

func Run() Report {
	cfg, _ := configLoadFn()

	var results []CheckResult
	results = append(results, checkCACert())
	results = append(results, checkCATrust())
	results = append(results, checkPortForwarding())

	if cfg != nil {
		for _, d := range cfg.Domains {
			results = append(results, checkHostsFile(d.ResolvedHostname()))
		}
	}

	results = append(results, checkDaemon())

	if cfg != nil {
		for _, d := range cfg.Domains {
			results = append(results, checkLeafCert(d.ResolvedHostname()))
		}
	}

	return Report{Results: results}
}

func checkCACert() CheckResult {
	name := "CA certificate"

	data, err := readFileFn(cert.CACertPath())
	if err != nil {
		return CheckResult{Name: name, Status: Fail, Message: "not found"}
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return CheckResult{Name: name, Status: Fail, Message: "invalid PEM"}
	}

	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return CheckResult{Name: name, Status: Fail, Message: "cannot parse: " + err.Error()}
	}

	remaining := time.Until(c.NotAfter)
	if remaining <= 0 {
		return CheckResult{Name: name, Status: Fail, Message: "expired"}
	}
	if remaining < 30*24*time.Hour {
		return CheckResult{Name: name, Status: Warn, Message: fmt.Sprintf("expires soon (%s)", c.NotAfter.Format("2006-01-02"))}
	}

	return CheckResult{Name: name, Status: Pass, Message: fmt.Sprintf("valid, expires %s", c.NotAfter.Format("2006-01-02"))}
}

func checkCATrust() CheckResult {
	return verifyCAIsTrusted()
}

func checkPortForwarding() CheckResult {
	name := "Port forwarding"
	pf := newPortFwdFn()
	if !pf.IsEnabled() {
		return CheckResult{Name: name, Status: Warn, Message: "not configured"}
	}
	if !pf.IsLoaded() {
		if daemonIsRunningFn() {
			return CheckResult{Name: name, Status: Fail, Message: "configured but inactive (run: sudo pfctl -e && sudo pfctl -f /etc/pf.conf)"}
		}
		return CheckResult{Name: name, Status: Warn, Message: "configured but inactive (run: sudo pfctl -e && sudo pfctl -f /etc/pf.conf)"}
	}
	if daemonIsRunningFn() {
		missing := missingIngressPorts()
		if len(missing) > 0 {
			return CheckResult{
				Name:    name,
				Status:  Fail,
				Message: fmt.Sprintf("configured but local ingress is down on %s (run: sudo pfctl -e && sudo pfctl -f /etc/pf.conf)", strings.Join(missing, ", ")),
			}
		}
	}
	return CheckResult{Name: name, Status: Pass, Message: fmt.Sprintf("active (80→%d, 443→%d)", config.ProxyHTTPPort, config.ProxyHTTPSPort)}
}

func missingIngressPorts() []string {
	ports := []int{80, 443}
	var missing []string
	for _, port := range ports {
		conn, err := dialTimeoutFn("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
		if err != nil {
			missing = append(missing, fmt.Sprintf("%d", port))
			continue
		}
		_ = conn.Close()
	}
	return missing
}

func checkHostsFile(hostname string) CheckResult {
	name := "Hosts: " + hostname

	content, err := readFileFn("/etc/hosts")
	if err != nil {
		return CheckResult{Name: name, Status: Fail, Message: "cannot read /etc/hosts"}
	}

	if system.HasMarkedEntry(string(content), hostname) {
		return CheckResult{Name: name, Status: Pass, Message: "present in /etc/hosts"}
	}
	return CheckResult{Name: name, Status: Fail, Message: "missing from /etc/hosts"}
}

func checkDaemon() CheckResult {
	name := "Daemon"
	if !daemonIsRunningFn() {
		return CheckResult{Name: name, Status: Warn, Message: "not running"}
	}

	resp, err := daemonSendIPCFn(daemon.Request{Type: daemon.MsgStatus})
	if err != nil || !resp.OK {
		return CheckResult{Name: name, Status: Fail, Message: "running but IPC failed"}
	}

	return CheckResult{Name: name, Status: Pass, Message: "running"}
}

func checkLeafCert(hostname string) CheckResult {
	name := "Cert: " + hostname

	data, err := readFileFn(cert.LeafCertPath(hostname))
	if err != nil {
		return CheckResult{Name: name, Status: Fail, Message: "not found"}
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return CheckResult{Name: name, Status: Fail, Message: "invalid PEM"}
	}

	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return CheckResult{Name: name, Status: Fail, Message: "cannot parse"}
	}

	remaining := time.Until(c.NotAfter)
	if remaining <= 0 {
		return CheckResult{Name: name, Status: Fail, Message: "expired"}
	}
	if remaining < 30*24*time.Hour {
		return CheckResult{Name: name, Status: Warn, Message: fmt.Sprintf("expires soon (%s)", c.NotAfter.Format("2006-01-02"))}
	}

	return CheckResult{Name: name, Status: Pass, Message: fmt.Sprintf("valid, expires %s", c.NotAfter.Format("2006-01-02"))}
}
