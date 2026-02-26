//go:build darwin

package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kamranahmedse/slim/internal/config"
)

const anchorName = "com.slim"
const anchorFile = "/etc/pf.anchors/com.slim"

var pfRules = fmt.Sprintf("rdr pass on lo0 inet proto tcp from any to 127.0.0.1 port 80 -> 127.0.0.1 port %d\nrdr pass on lo0 inet proto tcp from any to 127.0.0.1 port 443 -> 127.0.0.1 port %d\n",
	config.ProxyHTTPPort, config.ProxyHTTPSPort)

type darwinPortFwd struct{}

func NewPortForwarder() PortForwarder {
	return &darwinPortFwd{}
}

func (d *darwinPortFwd) Enable() error {
	if err := writeFileElevated(anchorFile, pfRules); err != nil {
		return fmt.Errorf("writing pf anchor: %w", err)
	}

	pfConf, err := os.ReadFile("/etc/pf.conf")
	if err != nil {
		return fmt.Errorf("reading pf.conf: %w", err)
	}

	conf := string(pfConf)
	anchorLoad := fmt.Sprintf("rdr-anchor \"%s\"", anchorName)
	anchorRule := fmt.Sprintf("load anchor \"%s\" from \"%s\"", anchorName, anchorFile)

	needsUpdate := false
	if !strings.Contains(conf, anchorLoad) {
		lines := strings.Split(conf, "\n")
		var updated []string
		inserted := false
		for _, line := range lines {
			updated = append(updated, line)
			if !inserted && strings.HasPrefix(line, "rdr-anchor") {
				updated = append(updated, anchorLoad)
				inserted = true
			}
		}
		if !inserted {
			updated = append([]string{anchorLoad}, updated...)
		}
		conf = strings.Join(updated, "\n")
		needsUpdate = true
	}
	if !strings.Contains(conf, anchorRule) {
		conf = strings.TrimRight(conf, "\n") + "\n" + anchorRule + "\n"
		needsUpdate = true
	}

	if needsUpdate {
		if err := writeFileElevated("/etc/pf.conf", conf); err != nil {
			return fmt.Errorf("writing pf.conf: %w", err)
		}
	}

	if output, err := exec.Command("sudo", "pfctl", "-e").CombinedOutput(); err != nil {
		out := strings.TrimSpace(string(output))
		if !isPFAlreadyEnabledOutput(out) {
			return fmt.Errorf("enabling pfctl: %s: %w", out, err)
		}
	}

	cmd := exec.Command("sudo", "pfctl", "-f", "/etc/pf.conf")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("loading pfctl rules: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

func (d *darwinPortFwd) Disable() error {
	if output, err := exec.Command("sudo", "rm", "-f", anchorFile).CombinedOutput(); err != nil {
		return fmt.Errorf("removing pf anchor: %s: %w", strings.TrimSpace(string(output)), err)
	}

	pfConf, err := os.ReadFile("/etc/pf.conf")
	if err != nil {
		return nil
	}

	conf := string(pfConf)
	anchorLoad := fmt.Sprintf("rdr-anchor \"%s\"", anchorName)
	anchorRule := fmt.Sprintf("load anchor \"%s\" from \"%s\"", anchorName, anchorFile)

	conf = strings.ReplaceAll(conf, anchorLoad+"\n", "")
	conf = strings.ReplaceAll(conf, anchorRule+"\n", "")

	if err := writeFileElevated("/etc/pf.conf", conf); err != nil {
		return fmt.Errorf("writing pf.conf: %w", err)
	}

	cmd := exec.Command("sudo", "pfctl", "-f", "/etc/pf.conf")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("reloading pfctl: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

func (d *darwinPortFwd) IsEnabled() bool {
	_, err := os.Stat(anchorFile)
	return err == nil
}

func isPFAlreadyEnabledOutput(out string) bool {
	return strings.Contains(strings.ToLower(out), "pf already enabled")
}
