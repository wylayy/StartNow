package installer

import (
	"fmt"
	"os"
	"os/exec"
)

const sudoPrompt = "password: "

func (e *Env) IsRoot() bool {
	return os.Geteuid() == 0
}

// SudoCommand returns the prefix to prepend to privileged commands: nil when
// running as root, otherwise ["sudo", "-p", prompt]. It errors when sudo is
// not installed.
func (e *Env) SudoCommand() ([]string, error) {
	if e.IsRoot() {
		return nil, nil
	}
	if _, err := exec.LookPath("sudo"); err != nil {
		return nil, fmt.Errorf("sudo not found — run startnow as root instead")
	}
	return []string{"sudo", "-p", sudoPrompt}, nil
}

func (e *Env) sudoStatus() string {
	if e.IsRoot() {
		return "root"
	}
	if _, err := exec.LookPath("sudo"); err != nil {
		return "missing"
	}
	if exec.Command("sudo", "-n", "true").Run() == nil {
		return "sudo (no password)"
	}
	return "sudo (password)"
}
