package supervisor

import (
	"chronosync.local/engine/internal/policy"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicySentinelChild(t *testing.T) {
	if path := os.Getenv("CHRONOSYNC_POLICY_TEST_SENTINEL"); path != "" {
		if err := os.WriteFile(path, []byte("launched"), 0600); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	}
}
func TestPolicyRefusedBeforeLaunch(t *testing.T) {
	p := policy.RequestedPolicy{Version: policy.Version, ID: "synthetic", Scope: policy.Scope{WorkspaceDigest: strings.Repeat("a", 64), ExecutableDigest: strings.Repeat("b", 64)}}
	for _, dim := range policy.Dimensions() {
		r := policy.Requirement{Dimension: dim, Mode: policy.Deny}
		switch dim {
		case policy.Workspace:
			r.Mode = policy.ReadOnly
		case policy.Environment:
			r.Mode = policy.Allowlist
		case policy.CPU, policy.Memory, policy.Disk, policy.ProcessCount, policy.Output, policy.Timeout, policy.Cancellation:
			r.Mode = policy.Bounded
			r.Limit = 1
		case policy.Approval, policy.Audit:
			r.Mode = policy.Required
		}
		p.Requirements = append(p.Requirements, r)
	}
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "well-formed", true: "malformed"}[invalid], func(t *testing.T) {
			request := p
			if invalid {
				request.Version = 0
			}
			marker := filepath.Join(t.TempDir(), "must-not-exist")
			called := false
			spec := Spec{Policy: &request, Path: os.Args[0], Args: []string{"-test.run=TestPolicySentinelChild"}, Env: append(os.Environ(), "CHRONOSYNC_POLICY_TEST_SENTINEL="+marker)}
			result := Run(context.Background(), spec, func([]byte) error { called = true; return nil })
			var refusal *PolicyError
			if !errors.As(result.Err, &refusal) || result.ExitCode != -1 || refusal.Decision.Outcome != policy.Refused || called {
				t.Fatalf("%+v", result)
			}
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("policy child launched: %v", err)
			}
			if !invalid && len(refusal.Decision.Reasons) != len(policy.Dimensions()) {
				t.Fatal(refusal.Decision)
			}
		})
	}
}
