package supervisor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestChildFixture(t *testing.T) {
	mode := os.Getenv("CHRONOSYNC_TEST_CHILD")
	if mode == "" {
		return
	}
	switch mode {
	case "success":
		fmt.Println(`{"type":"turn.completed"}`)
	case "failure":
		fmt.Println("partial")
		os.Exit(7)
	case "sleep":
		time.Sleep(30 * time.Second)
	case "flood":
		fmt.Print(strings.Repeat("x", MaxLine+1))
	case "grandchild":
		cmd := exec.Command(os.Args[0], "-test.run=TestChildFixture")
		cmd.Env = append(os.Environ(), "CHRONOSYNC_TEST_CHILD=sleep")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			os.Exit(2)
		}
		fmt.Println(cmd.Process.Pid)
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}
func child(mode string) Spec {
	return Spec{Path: os.Args[0], Args: []string{"-test.run=TestChildFixture"}, Env: append(os.Environ(), "CHRONOSYNC_TEST_CHILD="+mode)}
}
func TestProcessOutcomes(t *testing.T) {
	if !Supported() {
		t.Skip("platform gate")
	}
	for _, tc := range []struct {
		mode string
		code int
		fail bool
	}{{"success", 0, false}, {"failure", 7, true}, {"flood", -1, true}} {
		t.Run(tc.mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			var lines []string
			got := Run(ctx, child(tc.mode), func(b []byte) error { lines = append(lines, string(b)); return nil })
			if (got.Err != nil) != tc.fail {
				t.Fatalf("%+v", got)
			}
			if tc.mode == "failure" && (got.ExitCode != 7 || len(lines) != 1) {
				t.Fatal("lost partial history/exit")
			}
		})
	}
}
func TestDeadlineAndCancel(t *testing.T) {
	if !Supported() {
		t.Skip("platform gate")
	}
	for _, user := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		if user {
			go func() { time.Sleep(50 * time.Millisecond); cancel() }()
		}
		start := time.Now()
		got := Run(ctx, child("sleep"), func([]byte) error { return nil })
		cancel()
		if got.Err == nil || time.Since(start) > 2*time.Second {
			t.Fatalf("cleanup failed: %+v", got)
		}
	}
}
