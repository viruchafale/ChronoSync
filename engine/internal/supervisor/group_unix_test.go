//go:build darwin || linux

package supervisor

import (
	"context"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestGrandchildCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pid := 0
	got := Run(ctx, child("grandchild"), func(b []byte) error { pid, _ = strconv.Atoi(strings.TrimSpace(string(b))); cancel(); return nil })
	if got.Err == nil || pid == 0 {
		t.Fatalf("no fixture child: %+v", got)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) == syscall.ESRCH {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("grandchild still exists after cancellation")
}
