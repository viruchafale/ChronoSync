package analysis

import (
	"fmt"
	"strconv"
)

func DefaultPolicy() Policy {
	return Policy{
		FormatVersion: FormatVersion, Name: "strict-observable-v1", ExactMinEventAgreement: 1,
		PartialMinEventAgreement: 0.8, MinDimensionAgreement: 1, MaxMissingEvents: 0,
		MaxAdditionalEvents: 0, MaxReorderedEvents: 0, MaxChangedDimensions: 0,
		RequiredDimensions: []Dimension{EventType, Ordering}, UnsupportedHandling: "incomparable",
	}
}

func Evaluate(experiment Experiment, policy Policy) (Evaluation, error) {
	result := Evaluation{FormatVersion: FormatVersion, Decision: "fail", Passed: false, Rules: []RuleResult{}}
	if err := validatePolicy(policy); err != nil {
		return result, err
	}
	if experiment.FormatVersion != FormatVersion || len(experiment.Observations) < 2 || len(experiment.Comparisons) != len(experiment.Observations)-1 {
		return result, &Error{"INVALID_EXPERIMENT", "version or observation/comparison count"}
	}
	incomparable := false
	allPassed := true
	for _, dimension := range policy.RequiredDimensions {
		available := true
		for _, observation := range experiment.Observations {
			if !hasDimension(observation, dimension) {
				available = false
			}
		}
		rule := RuleResult{Rule: "required_dimension:" + string(dimension), Passed: available, Actual: strconv.FormatBool(available), Required: "true", Reason: "Required evidence must be present in every preserved observation."}
		result.Rules = append(result.Rules, rule)
		if !available {
			allPassed = false
			incomparable = true
		}
	}
	for index, comparison := range experiment.Comparisons {
		if comparison.Status == Unsupported || comparison.Status == Insufficient {
			incomparable = true
			allPassed = false
		}
		minimum := policy.PartialMinEventAgreement
		label := "partial"
		if comparison.Status == Exact {
			minimum = policy.ExactMinEventAgreement
			label = "exact"
		}
		prefix := fmt.Sprintf("comparison_%d:", index+1)
		checks := []RuleResult{
			thresholdRule(prefix+label+"_event_agreement", comparison.Metrics.EventAgreement >= minimum, comparison.Metrics.EventAgreement, minimum),
			thresholdRule(prefix+"dimension_agreement", comparison.Metrics.DimensionAgreement >= policy.MinDimensionAgreement, comparison.Metrics.DimensionAgreement, policy.MinDimensionAgreement),
			countRule(prefix+"missing_events", comparison.Metrics.MissingEvents <= policy.MaxMissingEvents, comparison.Metrics.MissingEvents, policy.MaxMissingEvents),
			countRule(prefix+"additional_events", comparison.Metrics.AdditionalEvents <= policy.MaxAdditionalEvents, comparison.Metrics.AdditionalEvents, policy.MaxAdditionalEvents),
			countRule(prefix+"reordered_events", comparison.Metrics.ReorderedEvents <= policy.MaxReorderedEvents, comparison.Metrics.ReorderedEvents, policy.MaxReorderedEvents),
			countRule(prefix+"changed_dimensions", comparison.Metrics.ChangedDimensions <= policy.MaxChangedDimensions, comparison.Metrics.ChangedDimensions, policy.MaxChangedDimensions),
		}
		for _, check := range checks {
			result.Rules = append(result.Rules, check)
			if !check.Passed {
				allPassed = false
			}
		}
	}
	if incomparable && policy.UnsupportedHandling == "incomparable" {
		result.Decision = "incomparable"
		result.Summary = "Required or supported evidence is unavailable; the policy did not pass."
		return result, nil
	}
	if allPassed {
		result.Decision = "pass"
		result.Passed = true
		result.Summary = "Every explicit policy rule passed for every preserved comparison."
	} else {
		result.Summary = "One or more explicit policy rules failed."
	}
	return result, nil
}

func validatePolicy(policy Policy) error {
	if policy.FormatVersion != FormatVersion || policy.Name == "" || len(policy.Name) > 128 || policy.ExactMinEventAgreement < 0 || policy.ExactMinEventAgreement > 1 || policy.PartialMinEventAgreement < 0 || policy.PartialMinEventAgreement > 1 || policy.ExactMinEventAgreement < policy.PartialMinEventAgreement || policy.MinDimensionAgreement < 0 || policy.MinDimensionAgreement > 1 || policy.MaxMissingEvents < 0 || policy.MaxAdditionalEvents < 0 || policy.MaxReorderedEvents < 0 || policy.MaxChangedDimensions < 0 || (policy.UnsupportedHandling != "fail" && policy.UnsupportedHandling != "incomparable") {
		return &Error{"INVALID_POLICY", "version, name, thresholds, limits, or unsupported handling"}
	}
	seen := map[Dimension]bool{}
	for _, required := range policy.RequiredDimensions {
		known := false
		for _, dimension := range dimensions {
			known = known || required == dimension
		}
		if !known || seen[required] {
			return &Error{"INVALID_POLICY", "unknown or duplicate required dimension"}
		}
		seen[required] = true
	}
	return nil
}

func hasDimension(observation Observation, dimension Dimension) bool {
	for _, gap := range append(append([]Gap{}, observation.Missing...), observation.Unsupported...) {
		if gap.Dimension == dimension {
			return false
		}
	}
	switch dimension {
	case EventType, Ordering, Config:
		return len(observation.Events) > 0
	case Snapshot:
		return len(observation.Snapshots) > 0
	}
	for _, event := range observation.Events {
		var value string
		switch dimension {
		case Input:
			value = event.InputSHA256
		case Output:
			value = event.OutputSHA256
		case Tool:
			value = event.ToolSHA256
		case Request:
			value = event.RequestSHA256
		case Response:
			value = event.ResponseSHA256
		case Workspace:
			value = event.WorkspaceSHA256
		}
		if value != "" {
			return true
		}
	}
	return false
}

func thresholdRule(name string, passed bool, actual, required float64) RuleResult {
	return RuleResult{Rule: name, Passed: passed, Actual: strconv.FormatFloat(actual, 'f', 6, 64), Required: ">=" + strconv.FormatFloat(required, 'f', 6, 64), Reason: "The metric is calculated from retained individual evidence."}
}

func countRule(name string, passed bool, actual, required int) RuleResult {
	return RuleResult{Rule: name, Passed: passed, Actual: strconv.Itoa(actual), Required: "<=" + strconv.Itoa(required), Reason: "Observed divergence count must not exceed the configured maximum."}
}
