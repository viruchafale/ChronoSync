//go:build darwin || linux

package snapshot_test

import (
	"chronosync.local/engine/internal/snapshot"
	"context"
	"path/filepath"
	"syscall"
	"testing"
)

func TestSpecialFileRejectedWithoutOpening(t *testing.T) {
	f := setup(t)
	if err := syscall.Mkfifo(filepath.Join(f.source, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	record, err := f.s.Capture(context.Background(), snapshot.CaptureSpec{RunID: f.run.ID, Consent: true})
	if err == nil || record.State != "failed" {
		t.Fatal(record, err)
	}
}
