//go:build darwin || linux

package supervisor

import (
	"os/exec"
	"syscall"
)

func Supported() bool   { return true }
func setup(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func signal(c *exec.Cmd, force bool) {
	if c.Process != nil {
		s := syscall.SIGTERM
		if force {
			s = syscall.SIGKILL
		}
		_ = syscall.Kill(-c.Process.Pid, s)
	}
}
