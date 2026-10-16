//go:build darwin || linux

package snapshot

import (
	"os"
	"syscall"
)

func openRegular(root *os.Root, p string) (*os.File, error) {
	return root.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
}
