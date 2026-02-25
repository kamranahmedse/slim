//go:build darwin

package portfwd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const anchorName = "com.localname"
const anchorFile = "/etc/pf.anchors/com.localname"

const pfRules = `rdr pass on lo0 inet proto tcp from any to 127.0.0.1 port 80 -> 127.0.0.1 port 10080
rdr pass on lo0 inet proto tcp from any to 127.0.0.1 port 443 -> 127.0.0.1 port 10443
`

type DarwinPortFwd struct{}

func New() PortForwarder {
	return &DarwinPortFwd{}
}

func (d *DarwinPortFwd) Enable() error {
	if err := sudoWrite(anchorFile, pfRules); err != nil {
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
		if err := sudoWrite("/etc/pf.conf", conf); err != nil {
			return fmt.Errorf("writing pf.conf: %w", err)
		}
	}

	cmd := exec.Command("sudo", "pfctl", "-ef", "/etc/pf.conf")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("enabling pfctl: %w", err)
	}

	return nil
}

func (d *DarwinPortFwd) Disable() error {
	exec.Command("sudo", "rm", "-f", anchorFile).Run()

	pfConf, err := os.ReadFile("/etc/pf.conf")
	if err != nil {
		return nil
	}

	conf := string(pfConf)
	anchorLoad := fmt.Sprintf("rdr-anchor \"%s\"", anchorName)
	anchorRule := fmt.Sprintf("load anchor \"%s\" from \"%s\"", anchorName, anchorFile)

	conf = strings.ReplaceAll(conf, anchorLoad+"\n", "")
	conf = strings.ReplaceAll(conf, anchorRule+"\n", "")

	sudoWrite("/etc/pf.conf", conf)
	exec.Command("sudo", "pfctl", "-ef", "/etc/pf.conf").Run()
	return nil
}

func (d *DarwinPortFwd) IsEnabled() bool {
	_, err := os.Stat(anchorFile)
	return err == nil
}

func sudoWrite(path string, content string) error {
	err := os.WriteFile(path, []byte(content), 0644)
	if err == nil {
		return nil
	}

	if !os.IsPermission(err) {
		return err
	}

	cmd := exec.Command("sudo", "tee", path)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
