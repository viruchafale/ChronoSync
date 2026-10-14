package codex

import (
	"chronosync.local/engine/internal/adapter"
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/supervisor"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type CLI struct{}

var versionPattern = regexp.MustCompile(`^codex-cli 0\.(\d+)\.(\d+)$`)

func Environment() []string {
	// Credentials stay owned by Codex; API-key/proxy/shell injection env is not inherited.
	out := []string{"PATH=/usr/local/bin:/usr/bin:/bin:/opt/homebrew/bin", "NO_COLOR=1"}
	for _, k := range []string{"HOME", "USER", "TMPDIR", "LANG", "LC_ALL", "CODEX_HOME"} {
		if v := os.Getenv(k); v != "" {
			out = append(out, k+"="+v)
		}
	}
	return out
}
func (CLI) Version(ctx context.Context, path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("Codex executable must be an absolute path")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var version string
	res := supervisor.Run(ctx, supervisor.Spec{Path: path, Args: []string{"--version"}, Env: Environment()}, func(b []byte) error { version = string(b); return nil })
	match := versionPattern.FindStringSubmatch(version)
	if res.Err != nil || len(match) != 3 {
		return "", errors.New("cannot verify Codex CLI version")
	}
	minor, _ := strconv.Atoi(match[1])
	if minor < 163 {
		return "", errors.New("Codex CLI >=0.163.0 required")
	}
	return version, nil
}
func (CLI) Record(ctx context.Context, spec model.StartSpec, sink adapter.Sink) adapter.Result {
	result := adapter.Result{ExitCode: -1}
	args := []string{"exec", "--json", "--sandbox", spec.Sandbox, "--ignore-user-config", "--ephemeral", "--color", "never", "-C", spec.Workspace, "-"}
	res := supervisor.Run(ctx, supervisor.Spec{Path: spec.CodexPath, Args: args, Dir: spec.Workspace, Env: Environment(), Input: spec.Prompt}, func(line []byte) error {
		rec, terminal, err := Parse(line)
		if err != nil {
			result.Malformed++
			return sink(adapter.Record{Kind: "adapter.warning", Payload: model.Payload{Message: "Malformed upstream record omitted"}})
		}
		if terminal == "completed" {
			result.Completed = true
		}
		if terminal == "failed" {
			result.Failed = true
		}
		return sink(rec)
	})
	result.ExitCode = res.ExitCode
	result.Diagnostic = res.Diagnostic
	result.Err = res.Err
	return result
}

type item struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Text     string `json:"text"`
	Command  string `json:"command"`
	Output   string `json:"aggregated_output"`
	ExitCode *int   `json:"exit_code"`
	Changes  []struct {
		Path string `json:"path"`
		Kind string `json:"kind"`
	} `json:"changes"`
}
type wire struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Item     *item  `json:"item"`
	Message  string `json:"message"`
	Error    *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func Parse(b []byte) (adapter.Record, string, error) {
	var w wire
	if len(b) > supervisor.MaxLine || json.Unmarshal(b, &w) != nil || w.Type == "" {
		return adapter.Record{}, "", errors.New("invalid Codex record")
	}
	p := model.Payload{UpstreamType: w.Type}
	kind := "adapter.unknown"
	terminal := ""
	switch w.Type {
	case "thread.started":
		if w.ThreadID == "" {
			return adapter.Record{}, "", errors.New("missing thread id")
		}
		kind = "agent.thread"
		p.ThreadID = w.ThreadID
	case "turn.started":
		kind = "agent.turn_started"
	case "turn.completed":
		kind = "agent.turn_completed"
		terminal = "completed"
	case "turn.failed":
		kind = "agent.turn_failed"
		terminal = "failed"
		if w.Error != nil {
			p.Message = w.Error.Message
		}
	case "error":
		kind = "adapter.error"
		terminal = "failed"
		p.Message = w.Message
	case "item.started", "item.updated", "item.completed":
		if w.Item == nil || w.Item.ID == "" || w.Item.Type == "" {
			return adapter.Record{}, "", errors.New("invalid item")
		}
		i := w.Item
		p.ItemID = i.ID
		p.UpstreamType = w.Type + ":" + i.Type
		suffix := strings.TrimPrefix(w.Type, "item.")
		switch i.Type {
		case "agent_message":
			kind = "agent.message"
			p.Message = i.Text
		case "reasoning":
			kind = "agent.reasoning_summary"
			p.Message = i.Text
		case "command_execution":
			if i.Command == "" {
				return adapter.Record{}, "", errors.New("missing command")
			}
			kind = "command." + suffix
			p.Command = i.Command
			p.Output = i.Output
			p.ExitCode = i.ExitCode
		case "mcp_tool_call", "web_search":
			kind = "tool." + suffix
			p.Message = "Observable tool lifecycle; raw arguments/results omitted"
		case "file_change":
			kind = "file.reported"
			p.Source = "codex"
			if len(i.Changes) > 1000 {
				return adapter.Record{}, "", errors.New("too many file changes")
			}
			for _, c := range i.Changes {
				p.Changes = append(p.Changes, model.Change{Path: c.Path, Operation: c.Kind})
			}
		}
	}
	if kind == "adapter.unknown" {
		p.Message = "Unknown upstream type; raw payload omitted"
	}
	return adapter.Record{Kind: kind, Payload: p}, terminal, nil
}
