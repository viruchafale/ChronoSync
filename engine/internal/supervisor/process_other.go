//go:build !darwin && !linux

package supervisor

import "os/exec"

func Supported() bool                { return false }
func setup(c *exec.Cmd)              {}
func signal(c *exec.Cmd, force bool) {}
