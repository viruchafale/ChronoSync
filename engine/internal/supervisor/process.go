// Package supervisor bounds processes and their captured output. It is not a sandbox.
package supervisor

import (
	"bufio"
	"chronosync.local/engine/internal/policy"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const MaxLine = 128 * 1024
const MaxOutput = 8 * 1024 * 1024
const Grace = 500 * time.Millisecond

type Spec struct {
	// Policy opts into fail-closed policy admission. Nil retains legacy execution
	// semantics and must never be presented as evaluated or isolated.
	Policy *policy.RequestedPolicy
	Path   string
	Args   []string
	Dir    string
	Env    []string
	Input  string
}
type Result struct {
	ExitCode   int
	Diagnostic string
	Err        error
}

func Run(parent context.Context, spec Spec, onLine func([]byte) error) Result {
	if spec.Policy != nil {
		return Result{ExitCode: -1, Err: refusePolicy(*spec.Policy)}
	}
	if !Supported() {
		return Result{-1, "", errors.New("recording unsupported on this OS: process-tree containment unavailable")}
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	cmd := exec.Command(spec.Path, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	setup(cmd)
	stdin, input, err := os.Pipe()
	if err != nil {
		return Result{-1, "", err}
	}
	defer stdin.Close()
	defer input.Close()
	stdout, out, err := os.Pipe()
	if err != nil {
		return Result{-1, "", err}
	}
	defer stdout.Close()
	defer out.Close()
	stderr, errOut, err := os.Pipe()
	if err != nil {
		return Result{-1, "", err}
	}
	defer stderr.Close()
	defer errOut.Close()
	cmd.Stdin = stdin
	cmd.Stdout = out
	cmd.Stderr = errOut
	if err = cmd.Start(); err != nil {
		return Result{-1, "", err}
	}
	_ = stdin.Close()
	_ = out.Close()
	_ = errOut.Close()
	inputDone := make(chan struct{})
	go func() { defer close(inputDone); _, _ = io.WriteString(input, spec.Input); _ = input.Close() }()
	done := make(chan struct{})
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		select {
		case <-ctx.Done():
			signal(cmd, false)
			select {
			case <-time.After(Grace):
				signal(cmd, true)
			case <-done:
			}
		case <-done:
		}
	}()
	var wg sync.WaitGroup
	wg.Add(2)
	var total atomic.Int64
	var firstErr error
	var mu sync.Mutex
	fail := func(e error) {
		if e != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = e
			}
			mu.Unlock()
			cancel()
		}
	}
	go func() {
		defer wg.Done()
		s := bufio.NewScanner(stdout)
		s.Buffer(make([]byte, 4096), MaxLine)
		for s.Scan() {
			if total.Add(int64(len(s.Bytes())+1)) > MaxOutput {
				fail(errors.New("output budget exceeded"))
				return
			}
			if e := onLine(s.Bytes()); e != nil {
				fail(e)
				return
			}
		}
		fail(s.Err())
	}()
	var diagnostics strings.Builder
	go func() {
		defer wg.Done()
		b := make([]byte, 4096)
		for {
			n, e := stderr.Read(b)
			if n > 0 {
				if total.Add(int64(n)) > MaxOutput {
					fail(errors.New("output budget exceeded"))
					return
				}
				if diagnostics.Len() < 32*1024 {
					keep := min(n, 32*1024-diagnostics.Len())
					diagnostics.Write(b[:keep])
				}
			}
			if e != nil {
				if e != io.EOF {
					fail(e)
				}
				return
			}
		}
	}()
	waitErr := cmd.Wait()
	signal(cmd, true) // normal exit must not leave ordinary descendants holding pipes
	readersDone := make(chan struct{})
	go func() { wg.Wait(); close(readersDone) }()
	select {
	case <-readersDone:
	case <-time.After(time.Second):
		_ = stdout.Close()
		_ = stderr.Close()
		<-readersDone
		fail(errors.New("output pipes remained open after process exit"))
	}
	_ = input.Close()
	close(done)
	<-watchDone
	<-inputDone
	code := -1
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	if firstErr != nil {
		err = firstErr
	} else if parent.Err() != nil {
		err = parent.Err()
	} else {
		err = waitErr
	}
	return Result{code, diagnostics.String(), err}
}
