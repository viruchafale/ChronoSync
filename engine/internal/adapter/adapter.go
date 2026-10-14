package adapter

import (
	"chronosync.local/engine/internal/model"
	"context"
)

type Record struct {
	Kind    string
	Payload model.Payload
}
type Result struct {
	ExitCode   int
	Completed  bool
	Failed     bool
	Malformed  int
	Diagnostic string
	Err        error
}
type Sink func(Record) error
type Adapter interface {
	Version(context.Context, string) (string, error)
	Record(context.Context, model.StartSpec, Sink) Result
}
