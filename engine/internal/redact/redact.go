package redact

import (
	"chronosync.local/engine/internal/model"
	"regexp"
	"strings"
)

var patterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*(?:bearer\s+)?)[^\s,;"']+`),
	regexp.MustCompile(`(?i)((?:api[_-]?key|access[_-]?token|refresh[_-]?token|password|secret)\s*[=:]\s*["']?)[^\s,;"']+`),
	regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{10,}|gh[pousr]_[A-Za-z0-9]{10,}|AKIA[A-Z0-9]{16})\b`),
	regexp.MustCompile(`-----BEGIN [^-]*PRIVATE KEY-----[\s\S]*?(?:-----END [^-]*PRIVATE KEY-----|$)`),
}

func sanitize(s string) string {
	for i, p := range patterns {
		if i < 2 {
			s = p.ReplaceAllString(s, "${1}[REDACTED]")
		} else {
			s = p.ReplaceAllString(s, "[REDACTED]")
		}
	}
	return s
}
func Text(s string) string {
	s = sanitize(s)
	const max = 32 * 1024
	if len(s) > max {
		s = strings.ToValidUTF8(s[:max], "�") + " [TRUNCATED]"
	}
	return s
}
func Payload(p model.Payload) model.Payload {
	for _, field := range []struct {
		name  string
		value string
	}{{"message", p.Message}, {"command", p.Command}, {"output", p.Output}, {"reason", p.Reason}, {"item_id", p.ItemID}, {"thread_id", p.ThreadID}, {"upstream_type", p.UpstreamType}, {"source", p.Source}} {
		value := sanitize(field.value)
		if len(value) > 32768 {
			p.Truncations = append(p.Truncations, model.Truncation{Field: field.name, OriginalBytes: len(value), LimitBytes: 32768})
		}
	}
	p.Source = Text(p.Source)
	p.Message = Text(p.Message)
	p.Command = Text(p.Command)
	p.Output = Text(p.Output)
	p.Reason = Text(p.Reason)
	p.ItemID = Text(p.ItemID)
	p.ThreadID = Text(p.ThreadID)
	p.UpstreamType = Text(p.UpstreamType)
	for i := range p.Changes {
		p.Changes[i].Path = Text(p.Changes[i].Path)
	}
	return p
}
