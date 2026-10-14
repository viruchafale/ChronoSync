package redact

import (
	"chronosync.local/engine/internal/model"
	"strings"
	"testing"
)

func TestSecrets(t *testing.T) {
	for _, text := range []string{"Authorization: Bearer synthetic-token", "api_key=synthetic-secret", "password=synthetic-secret", "sk-synthetic1234567890", "-----BEGIN PRIVATE KEY-----\nsynthetic\n-----END PRIVATE KEY-----"} {
		got := Text(text)
		if strings.Contains(got, "synthetic") {
			t.Fatalf("secret retained: %q", got)
		}
	}
}
func TestBound(t *testing.T) {
	if got := Text(strings.Repeat("a", 40000)); len(got) > 33000 || !strings.Contains(got, "TRUNCATED") {
		t.Fatal("missing bound")
	}
}

func TestTruncationMetadata(t *testing.T) {
	p := Payload(model.Payload{Output: strings.Repeat("x", 40000)})
	if len(p.Truncations) != 1 || p.Truncations[0].Field != "output" || p.Truncations[0].OriginalBytes != 40000 || p.Truncations[0].LimitBytes != 32768 || !strings.HasSuffix(p.Output, " [TRUNCATED]") {
		t.Fatal(p.Truncations)
	}
	redacted := Payload(model.Payload{Message: "api_key=" + strings.Repeat("x", 40000)})
	if len(redacted.Truncations) != 0 || !strings.Contains(redacted.Message, "[REDACTED]") {
		t.Fatal("redaction falsely reported as truncation")
	}
}
