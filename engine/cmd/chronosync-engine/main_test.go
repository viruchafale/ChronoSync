package main

import (
	"bytes"
	"chronosync.local/engine/internal/contract"
	"encoding/json"
	"errors"
	"testing"
)

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("closed pipe") }
func TestVersion(t *testing.T) {
	var out, stderr bytes.Buffer
	if code := run([]string{"version"}, &out, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, &stderr)
	}
	var result contract.Version
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ProtocolVersion != 1 || len(result.Capabilities) != 5 || result.Capabilities[0] != "version" {
		t.Fatalf("unsupported capability claim: %+v", result)
	}
}
func TestRejectUnimplementedCommands(t *testing.T) {
	for _, args := range [][]string{nil, {"record"}, {"version", "extra"}} {
		var out, stderr bytes.Buffer
		if code := run(args, &out, &stderr); code != 2 || out.Len() != 0 || stderr.Len() == 0 {
			t.Fatalf("invalid args accepted: %v", args)
		}
	}
}
func TestWriteFailure(t *testing.T) {
	var stderr bytes.Buffer
	if run([]string{"version"}, brokenWriter{}, &stderr) != 1 {
		t.Fatal("write failure must fail")
	}
}
