//go:build linux

package system

import (
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestLinuxPortForwardEnableTwiceInstallsOutputJumpOnce(t *testing.T) {
	restore := snapshotLinuxPortFwdHooks()
	defer restore()

	mock := &iptablesMock{}
	commandExistsLinuxFn = func(name string) bool { return name == "iptables" }
	runPrivilegedLinuxFn = mock.run

	pf := &linuxPortFwd{}
	if err := pf.Enable(); err != nil {
		t.Fatalf("first Enable: %v", err)
	}
	if err := pf.Enable(); err != nil {
		t.Fatalf("second Enable: %v", err)
	}

	if got := mock.countCommand("-I", "OUTPUT"); got != 1 {
		t.Fatalf("expected one OUTPUT jump insert, got %d", got)
	}
	if got := mock.countCommand("-A", linuxChainName); got != 4 {
		t.Fatalf("expected four redirect appends (2 per enable), got %d", got)
	}
}

func TestLinuxPortForwardDisableRemovesRules(t *testing.T) {
	restore := snapshotLinuxPortFwdHooks()
	defer restore()

	mock := &iptablesMock{
		chainExists: true,
		outputJump:  true,
	}
	commandExistsLinuxFn = func(name string) bool { return name == "iptables" }
	runPrivilegedLinuxFn = mock.run

	pf := &linuxPortFwd{}
	if err := pf.Disable(); err != nil {
		t.Fatalf("Disable: %v", err)
	}

	if got := mock.countCommand("-D", "OUTPUT"); got != 1 {
		t.Fatalf("expected one OUTPUT jump removal, got %d", got)
	}
	if got := mock.countCommand("-F", linuxChainName); got != 1 {
		t.Fatalf("expected one chain flush, got %d", got)
	}
	if got := mock.countCommand("-X", linuxChainName); got != 1 {
		t.Fatalf("expected one chain delete, got %d", got)
	}
}

func TestLinuxPortForwardEnableFailsWhenIptablesMissing(t *testing.T) {
	restore := snapshotLinuxPortFwdHooks()
	defer restore()

	commandExistsLinuxFn = func(string) bool { return false }
	pf := &linuxPortFwd{}

	if err := pf.Enable(); err == nil {
		t.Fatal("expected Enable to fail when iptables is missing")
	}
}

func TestLinuxPortForwardIsEnabledCheck(t *testing.T) {
	restore := snapshotLinuxPortFwdHooks()
	defer restore()

	commandExistsLinuxFn = func(name string) bool { return name == "iptables" }
	execCommandLinuxFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 0")
	}

	pf := &linuxPortFwd{}
	if !pf.IsEnabled() {
		t.Fatal("expected IsEnabled true when iptables check command succeeds")
	}

	execCommandLinuxFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 1")
	}
	if pf.IsEnabled() {
		t.Fatal("expected IsEnabled false when iptables check command fails")
	}
}

type iptablesMock struct {
	commands    [][]string
	chainExists bool
	outputJump  bool
}

func (m *iptablesMock) run(name string, args ...string) ([]byte, error) {
	cmd := append([]string{name}, args...)
	m.commands = append(m.commands, cmd)

	if len(args) < 4 || args[0] != "-t" || args[1] != "nat" {
		return []byte("invalid"), errors.New("invalid command")
	}

	switch {
	case matchPrefix(args, "-C", "OUTPUT"):
		if m.outputJump {
			return nil, nil
		}
		return []byte("Bad rule"), errors.New("exit 1")
	case matchPrefix(args, "-N", linuxChainName):
		if m.chainExists {
			return []byte("Chain already exists"), errors.New("exit 1")
		}
		m.chainExists = true
		return nil, nil
	case matchPrefix(args, "-F", linuxChainName):
		return nil, nil
	case matchPrefix(args, "-A", linuxChainName):
		return nil, nil
	case matchPrefix(args, "-I", "OUTPUT"):
		m.outputJump = true
		return nil, nil
	case matchPrefix(args, "-D", "OUTPUT"):
		if !m.outputJump {
			return []byte("Bad rule"), errors.New("exit 1")
		}
		m.outputJump = false
		return nil, nil
	case matchPrefix(args, "-X", linuxChainName):
		if !m.chainExists {
			return []byte("No chain/target/match by that name"), errors.New("exit 1")
		}
		m.chainExists = false
		return nil, nil
	default:
		return nil, nil
	}
}

func (m *iptablesMock) countCommand(action string, chain string) int {
	want := []string{action, chain}
	count := 0
	for _, cmd := range m.commands {
		joined := strings.Join(cmd, " ")
		if strings.Contains(joined, " "+want[0]+" "+want[1]) {
			count++
		}
	}
	return count
}

func matchPrefix(args []string, action string, chain string) bool {
	if len(args) < 4 {
		return false
	}
	return reflect.DeepEqual(args[2:4], []string{action, chain})
}

func snapshotLinuxPortFwdHooks() func() {
	prevExists := commandExistsLinuxFn
	prevRun := runPrivilegedLinuxFn
	prevExec := execCommandLinuxFn

	return func() {
		commandExistsLinuxFn = prevExists
		runPrivilegedLinuxFn = prevRun
		execCommandLinuxFn = prevExec
	}
}
