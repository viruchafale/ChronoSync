//go:build !darwin && !linux

package legacy

import (
	"errors"
	"os"
)

func lock(f *os.File) error { return errors.New("store unsupported on this OS") }
