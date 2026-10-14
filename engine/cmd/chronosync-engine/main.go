package main

import (
	"chronosync.local/engine/internal/adapter/codex"
	"chronosync.local/engine/internal/contract"
	"chronosync.local/engine/internal/ipc"
	"chronosync.local/engine/internal/objects"
	"chronosync.local/engine/internal/recorder"
	"chronosync.local/engine/internal/snapshot"
	"chronosync.local/engine/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func run(args []string, out, errOut io.Writer) int {
	if len(args) != 1 || args[0] != "version" {
		fmt.Fprintln(errOut, "usage: chronosync-engine version | serve [--data-dir ABSOLUTE_PRIVATE_DIRECTORY]")
		return 2
	}
	if err := json.NewEncoder(out).Encode(contract.CurrentVersion()); err != nil {
		fmt.Fprintln(errOut, "write version:", err)
		return 1
	}
	return 0
}
func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		os.Exit(serve(os.Args[2:]))
		return
	}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
func serve(args []string) int {
	var path string
	if len(args) == 2 && args[0] == "--data-dir" {
		path = args[1]
	} else if len(args) == 0 {
		base, err := os.UserConfigDir()
		if err != nil {
			return 1
		}
		path = filepath.Join(base, "ChronoSync", "recordings")
	} else {
		fmt.Fprintln(os.Stderr, "usage: chronosync-engine serve [--data-dir ABSOLUTE_PRIVATE_DIRECTORY]")
		return 2
	}
	store, err := storage.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer store.Close()
	if err = store.Recover(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	objectStore, err := objects.OpenIn(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "object store unavailable")
		return 1
	}
	defer objectStore.Close()
	snapshots, err := snapshot.New(store, objectStore, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "snapshot service unavailable")
		return 1
	}
	defer snapshots.Close()
	if err = snapshots.Recover(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "snapshot recovery failed")
		return 1
	}
	rec := recorder.New(store, codex.CLI{})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() { <-signals; rec.Close(); _ = os.Stdin.Close() }()
	if err = ipc.Serve(os.Stdin, os.Stdout, rec, store, snapshots); err != nil {
		fmt.Fprintln(os.Stderr, "engine protocol stopped:", err)
		return 1
	}
	return 0
}
