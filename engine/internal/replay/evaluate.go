package replay

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"chronosync.local/engine/internal/redact"
)

func Evaluate(ctx context.Context, raw []byte, evaluator Evaluator) (Result, error) {
	fixture, err := Load(raw)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		ReplayID:          fixture.ReplayID,
		SourceExecutionID: fixture.SourceExecutionID,
		FormatVersion:     fixture.FormatVersion,
		Status:            Completed,
		Mismatches:        []Mismatch{},
		Unsupported:       []UnsupportedInteraction{},
	}
	if evaluator == nil {
		result.Status = Failed
		result.Summary = "evaluation failed: evaluator unavailable"
		return result, nil
	}
	for _, interaction := range fixture.Interactions {
		if ctx.Err() != nil {
			result.Status = Cancelled
			result.Summary = "evaluation cancelled"
			return result, nil
		}
		actual, evalErr := evaluator.Evaluate(ctx, interaction.Input)
		if evalErr != nil {
			var unsupported *UnsupportedError
			if errors.As(evalErr, &unsupported) {
				result.Status = Unsupported
				result.Unsupported = append(result.Unsupported, UnsupportedInteraction{SourceSequence: interaction.SourceSequence, Reason: boundedReason(unsupported.Reason)})
				result.Summary = "evaluation stopped at an unsupported interaction"
				return result, nil
			}
			if errors.Is(evalErr, context.Canceled) || errors.Is(evalErr, context.DeadlineExceeded) || ctx.Err() != nil {
				result.Status = Cancelled
				result.Summary = "evaluation cancelled"
				return result, nil
			}
			result.Status = Failed
			result.Summary = "evaluation failed at source sequence " + interaction.SourceSequence
			return result, nil
		}
		if !validDigest(actual.OutputSHA256) {
			result.Status = Failed
			result.Summary = "evaluator returned an invalid output fingerprint"
			return result, nil
		}
		result.Evaluated++
		if actual.OutputSHA256 != interaction.Expected.OutputSHA256 {
			result.Mismatches = append(result.Mismatches, Mismatch{SourceSequence: interaction.SourceSequence, Field: "output_sha256", Expected: interaction.Expected.OutputSHA256, Actual: actual.OutputSHA256})
		}
		if actual.ExitCode != interaction.Expected.ExitCode {
			result.Mismatches = append(result.Mismatches, Mismatch{SourceSequence: interaction.SourceSequence, Field: "exit_code", Expected: strconv.Itoa(interaction.Expected.ExitCode), Actual: strconv.Itoa(actual.ExitCode)})
		}
	}
	result.Passed = len(result.Mismatches) == 0
	if result.Passed {
		result.Summary = "all supported interaction fingerprints matched"
	} else {
		result.Summary = strconv.Itoa(len(result.Mismatches)) + " regression mismatch(es)"
	}
	return result, nil
}

func boundedReason(reason string) string {
	if reason == "" {
		return "unsupported by evaluator"
	}
	reason = redact.Text(reason)
	if len(reason) > 256 {
		return strings.ToValidUTF8(reason[:256], "�")
	}
	return reason
}

// RegressionFixture returns canonical fixture bytes only after a successful,
// matching evaluation. The caller chooses an explicit private destination.
func RegressionFixture(raw []byte, result Result) ([]byte, error) {
	fixture, err := Load(raw)
	if err != nil {
		return nil, err
	}
	if result.ReplayID != fixture.ReplayID || result.SourceExecutionID != fixture.SourceExecutionID || result.FormatVersion != fixture.FormatVersion || result.Status != Completed || !result.Passed || result.Evaluated != len(fixture.Interactions) || len(result.Mismatches) != 0 || len(result.Unsupported) != 0 {
		return nil, typed("EVALUATION_NOT_PASSED", "regression fixture requires a complete matching evaluation")
	}
	return append([]byte(nil), raw...), nil
}
