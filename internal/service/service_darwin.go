//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const plistLabel = "com.localname.daemon"

type DarwinService struct{}

func New() Service {
	return &DarwinService{}
}

func plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", plistLabel+".plist")
}

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>up</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogDir}}/localname.log</string>
    <key>StandardErrorPath</key>
    <string>{{.LogDir}}/localname.log</string>
</dict>
</plist>
`

func (d *DarwinService) Install(binaryPath string) error {
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return err
	}

	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".localname", "log")
	os.MkdirAll(logDir, 0755)

	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(plistPath())
	if err != nil {
		return fmt.Errorf("creating plist: %w", err)
	}
	defer f.Close()

	err = tmpl.Execute(f, struct {
		Label      string
		BinaryPath string
		LogDir     string
	}{
		Label:      plistLabel,
		BinaryPath: absPath,
		LogDir:     logDir,
	})
	if err != nil {
		return err
	}

	cmd := exec.Command("launchctl", "load", plistPath())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("loading plist: %s: %w", string(output), err)
	}

	return nil
}

func (d *DarwinService) Uninstall() error {
	exec.Command("launchctl", "unload", plistPath()).Run()
	os.Remove(plistPath())
	return nil
}

func (d *DarwinService) IsInstalled() bool {
	_, err := os.Stat(plistPath())
	return err == nil
}
