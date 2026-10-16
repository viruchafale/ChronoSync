//go:build !darwin && !linux

package snapshot

import (
	"errors"
	"os"
)

func openRegular(root *os.Root, p string) (*os.File, error) {
	return nil, errors.New("snapshot filesystem unsupported")
}
