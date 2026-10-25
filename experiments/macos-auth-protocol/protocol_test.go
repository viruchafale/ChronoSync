package protocol

import (
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// No verifier exists outside this test file. This is not OS peer verification.
type syntheticPeerVerifier struct{ expected peerHandle }

func (v syntheticPeerVerifier) verify(p peerHandle) error {
	if p != v.expected {
		return errors.New("synthetic-private-error")
	}
	return nil
}
func fixture(t *testing.T) (*coordinator, challenge, peerHandle, *time.Time) {
	t.Helper()
	now := time.UnixMilli(1000)
	p := peerHandle{1}
	c, e := newCoordinator(func() time.Time { return now }, syntheticPeerVerifier{p})
	if e != nil {
		t.Fatal(e)
	}
	ch, e := c.begin(biometricPolicy, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	return c, ch, p, &now
}
func message(c challenge, outcome string) []byte {
	b, _ := json.Marshal(completion{c, outcome})
	return b
}
func TestCanonicalVector(t *testing.T) {
	// Independently reproduced with Python json/sha256, not a production manifest.
	c := challenge{1, strings.Repeat("1", 32), strings.Repeat("2", 32), manifestDigest(), strings.Repeat("3", 32), strings.Repeat("4", 32), 1000, 2000, biometricPolicy}
	if manifestDigest() != "ab77301ae655c1e6beca9bd002eb198db2d27676c637b9bd3db620e5a2d30f9d" {
		t.Fatal("manifest vector")
	}
	d, e := challengeDigest(c)
	if e != nil || d != "8d9faf8dca8ff1628be8b7964a691f247615d390c11ea4985b49431e364494b9" {
		t.Fatal(d, e)
	}
	for _, change := range []func(*challenge){func(c *challenge) { c.ID = strings.Repeat("5", 32) }, func(c *challenge) { c.Admission = strings.Repeat("5", 32) }, func(c *challenge) { c.Epoch = strings.Repeat("5", 32) }, func(c *challenge) { c.NativeSession = strings.Repeat("5", 32) }, func(c *challenge) { c.Expires++ }, func(c *challenge) { c.Policy = accountPolicy }} {
		other := c
		change(&other)
		got, e := challengeDigest(other)
		if e != nil || got == d {
			t.Fatal("unbound field", e)
		}
	}
	c.Manifest = strings.Repeat("a", 64)
	if _, e := canonical(c); e == nil {
		t.Fatal("arbitrary manifest")
	}
}
func TestWrongBindings(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*challenge)
	}{
		{"unknown", func(c *challenge) { c.ID = strings.Repeat("5", 32) }},
		{"admission", func(c *challenge) { c.Admission = strings.Repeat("5", 32) }},
		{"manifest", func(c *challenge) { c.Manifest = strings.Repeat("a", 64) }},
		{"epoch", func(c *challenge) { c.Epoch = strings.Repeat("5", 32) }},
		{"native-session", func(c *challenge) { c.NativeSession = strings.Repeat("5", 32) }},
		{"expiry", func(c *challenge) { c.Expires++ }},
		{"policy", func(c *challenge) { c.Policy = accountPolicy }},
		{"version", func(c *challenge) { c.Version = 99 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, ch, p, _ := fixture(t)
			tt.change(&ch)
			if _, e := c.complete(p, message(ch, "success")); e == nil {
				t.Fatal("mismatch accepted")
			}
		})
	}
}
func TestWrongAndUnverifiedPeers(t *testing.T) {
	c, ch, p, _ := fixture(t)
	if _, e := c.complete(peerHandle{2}, message(ch, "success")); e == nil || strings.Contains(e.Error(), "private") {
		t.Fatal("wrong peer/error disclosure", e)
	}
	c.verifier = nil
	if _, e := c.complete(p, message(ch, "success")); e == nil {
		t.Fatal("nil verifier trusted")
	}
}
func TestMalformedAndForgedFrames(t *testing.T) {
	_, ch, _, _ := fixture(t)
	valid := string(message(ch, "success"))
	for _, raw := range []string{"", "null", "[]", "{", strings.Repeat("x", maxMessage+1), valid + "{}", strings.Replace(valid, `"outcome":"success"`, `"outcome":"success","verified":true`, 1), strings.Replace(valid, `"outcome":"success"`, `"outcome":"denied","outcome":"success"`, 1), strings.Replace(valid, `"version":1`, `"version":"1"`, 1), strings.Replace(valid, `"success"`, `"synthetic-private-canary"`, 1)} {
		if _, e := decode([]byte(raw)); e == nil {
			t.Fatal("malformed frame decoded")
		}
		c, _, p, _ := fixture(t)
		if _, e := c.complete(p, []byte(raw)); e == nil || strings.Contains(e.Error(), "canary") {
			t.Fatal("malformed accepted or disclosed", e)
		}
	}
}
func TestCancelTimeoutDisconnectRestart(t *testing.T) {
	for _, mode := range []string{"cancel", "timeout", "disconnect", "clock-rollback"} {
		t.Run(mode, func(t *testing.T) {
			c, ch, p, now := fixture(t)
			switch mode {
			case "cancel":
				c.cancel()
			case "timeout":
				*now = now.Add(time.Second)
			case "disconnect":
				c.disconnect()
			case "clock-rollback":
				*now = now.Add(-time.Millisecond)
			}
			if _, e := c.complete(p, message(ch, "success")); e == nil {
				t.Fatal("late callback accepted")
			}
			if _, e := c.begin(biometricPolicy, time.Second); e == nil {
				t.Fatal("reopened terminal coordinator")
			}
		})
	}
	old, ch, p, _ := fixture(t)
	old.disconnect()
	fresh, _, _, _ := fixture(t)
	if _, e := fresh.complete(p, message(ch, "success")); e == nil {
		t.Fatal("replayed after restart")
	}
	if fresh.epoch == old.epoch || fresh.nativeSession == old.nativeSession {
		t.Fatal("session reused")
	}
}
func TestOneTimeCompletionAndTerminalOutcomes(t *testing.T) {
	for _, outcome := range []string{"success", "denied", "user_cancel", "system_cancel", "unavailable", "lockout", "invalid_context", "timeout"} {
		c, ch, p, _ := fixture(t)
		o, e := c.complete(p, message(ch, outcome))
		if e != nil || o.outcome != outcome {
			t.Fatal(o, e)
		}
		if _, e = c.complete(p, message(ch, "success")); e == nil {
			t.Fatal("reused outcome", outcome)
		}
	}
	c, ch, p, _ := fixture(t)
	start := make(chan struct{})
	results := make(chan error, 8)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; _, e := c.complete(p, message(ch, "success")); results <- e }()
	}
	close(start)
	wg.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal(success)
	}
}
func TestChallengeBoundsAndUnknownPolicy(t *testing.T) {
	for _, ttl := range []time.Duration{0, -time.Second, time.Minute + time.Millisecond} {
		c, _, _, _ := fixture(t)
		c.issued = false
		if _, e := c.begin(biometricPolicy, ttl); e == nil {
			t.Fatal("deadline")
		}
	}
	c, e := newCoordinator(time.Now, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.begin("unapproved-policy", time.Second); e == nil {
		t.Fatal("policy defaulted")
	}
}
func TestIsolationBoundary(t *testing.T) {
	// Reviewed import allowlist prevents this harness from acquiring a production
	// adapter/store/supervisor, network listener, filesystem writer or exec surface.
	allowed := map[string]bool{"bytes": true, "crypto/rand": true, "crypto/sha256": true, "encoding/hex": true, "encoding/json": true, "errors": true, "sync": true, "time": true}
	files, e := filepath.Glob("*.go")
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, e := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if e != nil {
			t.Fatal(e)
		}
		if f.Name.Name == "main" {
			t.Fatal("executable added")
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if !allowed[path] {
				t.Fatal("unreviewed import", path)
			}
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if ast.IsExported(d.Name.Name) {
					t.Fatal("exported protocol entry point")
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if typ, ok := spec.(*ast.TypeSpec); ok && ast.IsExported(typ.Name.Name) {
						t.Fatal("exported type")
					}
				}
			}
		}
	}
	module, e := os.ReadFile("go.mod")
	if e != nil || strings.Contains(string(module), "require") || strings.Contains(string(module), "replace") {
		t.Fatal("module dependency", e)
	}
}
