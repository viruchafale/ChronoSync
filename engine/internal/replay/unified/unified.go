package unified

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"chronosync.local/engine/internal/replay"
	"chronosync.local/engine/internal/replay/transport"
)

const (
	Version          = 1
	MaxPlanItems     = 10_000
	MaxTimeoutMillis = 30_000
)

type Actual struct {
	SourceSequence string `json:"source_sequence"`
	OutputSHA256   string `json:"output_sha256"`
	ExitCode       int    `json:"exit_code"`
}

type Plan struct {
	Version           int                          `json:"version"`
	TimeoutMillis     int                          `json:"timeout_millis"`
	Actuals           []Actual                     `json:"actuals"`
	TransportRequests []transport.SyntheticRequest `json:"transport_requests"`
}

type Status string

const (
	Completed   Status = "completed"
	Failed      Status = "failed"
	Cancelled   Status = "cancelled"
	Unsupported Status = "unsupported"
)

type Result struct {
	Version    int                          `json:"version"`
	Status     Status                       `json:"status"`
	Passed     bool                         `json:"passed"`
	Transport  []transport.TransportOutcome `json:"transport"`
	Regression replay.Result                `json:"regression"`
	Summary    string                       `json:"summary"`
}

type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Detail }

type actualEvaluator struct {
	actuals []Actual
	next    int
}

func (e *actualEvaluator) Evaluate(_ context.Context, input replay.Input) (replay.Actual, error) {
	if e.next >= len(e.actuals) || e.actuals[e.next].SourceSequence != input.SourceSequence {
		return replay.Actual{}, errors.New("controlled actual sequence mismatch")
	}
	actual := e.actuals[e.next]
	e.next++
	return replay.Actual{OutputSHA256: actual.OutputSHA256, ExitCode: actual.ExitCode}, nil
}

func Evaluate(ctx context.Context, regressionRaw, transportRaw []byte, plan Plan) (Result, error) {
	result := Result{Version: Version, Status: Failed, Transport: []transport.TransportOutcome{}}
	if plan.Version != Version || plan.TimeoutMillis < 1 || plan.TimeoutMillis > MaxTimeoutMillis || len(plan.Actuals) > MaxPlanItems || len(plan.TransportRequests) > MaxPlanItems {
		return result, &Error{"INVALID_PLAN", "version, timeout or item count"}
	}
	regressionFixture, err := replay.Load(regressionRaw)
	if err != nil {
		return result, &Error{"INVALID_REGRESSION_FIXTURE", "validation"}
	}
	transportRecording, err := transport.LoadTransportRecording(transportRaw)
	if err != nil {
		return result, &Error{"INVALID_TRANSPORT_FIXTURE", "validation"}
	}
	transportInspection, err := transport.Inspect(transportRaw)
	if err != nil || len(plan.TransportRequests) != transportInspection.Interactions || len(plan.Actuals) != len(regressionFixture.Interactions) {
		return result, &Error{"INVALID_PLAN", "plan does not cover fixture interactions"}
	}
	for index, actual := range plan.Actuals {
		if actual.SourceSequence != regressionFixture.Interactions[index].SourceSequence {
			return result, &Error{"INVALID_PLAN", "actual sequence"}
		}
	}
	evaluationContext, cancel := context.WithTimeout(ctx, time.Duration(plan.TimeoutMillis)*time.Millisecond)
	defer cancel()
	matcher := transport.NewTransportMatcher(transportRecording)
	for _, request := range plan.TransportRequests {
		outcome := matcher.Exchange(evaluationContext, request)
		result.Transport = append(result.Transport, outcome)
		switch outcome.Kind {
		case transport.TransportMatched:
			continue
		case transport.TransportCancelled:
			result.Status = Cancelled
			result.Summary = "evaluation cancelled"
		case transport.TransportUnsupported:
			result.Status = Unsupported
			result.Summary = "unsupported transport interaction"
		default:
			result.Status = Failed
			result.Summary = "transport interaction did not match"
		}
		return result, nil
	}
	evaluator := &actualEvaluator{actuals: append([]Actual(nil), plan.Actuals...)}
	regressionResult, err := replay.Evaluate(evaluationContext, regressionRaw, evaluator)
	if err != nil {
		return result, &Error{"EVALUATION_FAILED", "regression evaluation"}
	}
	result.Regression = regressionResult
	switch regressionResult.Status {
	case replay.Cancelled:
		result.Status = Cancelled
		result.Summary = "evaluation cancelled"
	case replay.Unsupported:
		result.Status = Unsupported
		result.Summary = "unsupported regression interaction"
	case replay.Completed:
		result.Status = Completed
		result.Passed = regressionResult.Passed
		if result.Passed {
			result.Summary = "all controlled interactions matched"
		} else {
			result.Summary = "regression mismatch"
		}
	default:
		result.Status = Failed
		result.Summary = "regression evaluation failed"
	}
	return result, nil
}

func MarshalResult(result Result) ([]byte, error) {
	if result.Version != Version || result.Status == "" || result.Transport == nil {
		return nil, &Error{"INVALID_RESULT", "result shape"}
	}
	return json.Marshal(result)
}

func LoadResult(raw []byte) (Result, error) {
	var result Result
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return Result{}, &Error{"INVALID_RESULT", "result encoding"}
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF || result.Version != Version || result.Status == "" || result.Transport == nil {
		return Result{}, &Error{"INVALID_RESULT", "result shape"}
	}
	canonical, err := json.Marshal(result)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Result{}, &Error{"INVALID_RESULT", "noncanonical result"}
	}
	return result, nil
}
