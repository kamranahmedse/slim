//go:build linux

package system

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kamranahmedse/localname/internal/config"
)

type linuxPortFwd struct{}

const linuxChainName = "LOCALNAME"

func NewPortForwarder() PortForwarder {
	return &linuxPortFwd{}
}

func (l *linuxPortFwd) Enable() error {
	if !commandExists("iptables") {
		return errors.New("iptables not found (install iptables)")
	}

	if err := l.ensureChain(); err != nil {
		return err
	}
	if err := l.ensureRedirectRule(80, config.ProxyHTTPPort); err != nil {
		return err
	}
	if err := l.ensureRedirectRule(443, config.ProxyHTTPSPort); err != nil {
		return err
	}

	exists, err := l.ruleExists("OUTPUT", "-o", "lo", "-p", "tcp", "-j", linuxChainName)
	if err != nil {
		return err
	}
	if !exists {
		if output, err := runPrivileged("iptables", "-t", "nat", "-I", "OUTPUT", "1", "-o", "lo", "-p", "tcp", "-j", linuxChainName); err != nil {
			return fmt.Errorf("installing OUTPUT jump rule: %s: %w", strings.TrimSpace(string(output)), err)
		}
	}
	return nil
}

func (l *linuxPortFwd) Disable() error {
	if !commandExists("iptables") {
		return nil
	}

	for {
		exists, err := l.ruleExists("OUTPUT", "-o", "lo", "-p", "tcp", "-j", linuxChainName)
		if err != nil {
			return err
		}
		if !exists {
			break
		}
		if output, err := runPrivileged("iptables", "-t", "nat", "-D", "OUTPUT", "-o", "lo", "-p", "tcp", "-j", linuxChainName); err != nil {
			return fmt.Errorf("removing OUTPUT jump rule: %s: %w", strings.TrimSpace(string(output)), err)
		}
	}

	if output, err := runPrivileged("iptables", "-t", "nat", "-F", linuxChainName); err != nil && !iptablesChainMissing(output) {
		return fmt.Errorf("flushing chain %s: %s: %w", linuxChainName, strings.TrimSpace(string(output)), err)
	}
	if output, err := runPrivileged("iptables", "-t", "nat", "-X", linuxChainName); err != nil && !iptablesChainMissing(output) {
		return fmt.Errorf("deleting chain %s: %s: %w", linuxChainName, strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (l *linuxPortFwd) IsEnabled() bool {
	if !commandExists("iptables") {
		return false
	}
	cmd := exec.Command("iptables", "-t", "nat", "-C", "OUTPUT", "-o", "lo", "-p", "tcp", "-j", linuxChainName)
	return cmd.Run() == nil
}

func (l *linuxPortFwd) ensureChain() error {
	if output, err := runPrivileged("iptables", "-t", "nat", "-N", linuxChainName); err != nil && !iptablesChainAlreadyExists(output) {
		return fmt.Errorf("creating chain %s: %s: %w", linuxChainName, strings.TrimSpace(string(output)), err)
	}
	if output, err := runPrivileged("iptables", "-t", "nat", "-F", linuxChainName); err != nil {
		return fmt.Errorf("flushing chain %s: %s: %w", linuxChainName, strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (l *linuxPortFwd) ensureRedirectRule(fromPort int, toPort int) error {
	args := []string{
		"-t", "nat",
		"-A", linuxChainName,
		"-p", "tcp",
		"-d", "127.0.0.1/32",
		"--dport", fmt.Sprintf("%d", fromPort),
		"-j", "REDIRECT",
		"--to-ports", fmt.Sprintf("%d", toPort),
	}
	if output, err := runPrivileged("iptables", args...); err != nil {
		return fmt.Errorf("adding redirect rule %d->%d: %s: %w", fromPort, toPort, strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (l *linuxPortFwd) ruleExists(chain string, ruleArgs ...string) (bool, error) {
	args := append([]string{"-t", "nat", "-C", chain}, ruleArgs...)
	output, err := runPrivileged("iptables", args...)
	if err == nil {
		return true, nil
	}
	msg := strings.ToLower(strings.TrimSpace(string(output)))
	if strings.Contains(msg, "bad rule") || strings.Contains(msg, "no chain/target/match by that name") || strings.Contains(msg, "does a matching rule exist") || strings.Contains(msg, "not found") {
		return false, nil
	}
	return false, fmt.Errorf("checking iptables rule: %s: %w", strings.TrimSpace(string(output)), err)
}

func iptablesChainAlreadyExists(output []byte) bool {
	msg := strings.ToLower(strings.TrimSpace(string(output)))
	return strings.Contains(msg, "chain already exists") || strings.Contains(msg, "file exists")
}

func iptablesChainMissing(output []byte) bool {
	msg := strings.ToLower(strings.TrimSpace(string(output)))
	return strings.Contains(msg, "no chain/target/match by that name") || strings.Contains(msg, "does a matching rule exist") || strings.Contains(msg, "not found")
}

func runPrivileged(name string, args ...string) ([]byte, error) {
	if os.Geteuid() == 0 {
		return exec.Command(name, args...).CombinedOutput()
	}
	all := append([]string{name}, args...)
	return exec.Command("sudo", all...).CombinedOutput()
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
