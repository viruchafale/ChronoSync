package analysis

import "context"

func Run(ctx context.Context, observer Observer, runIDs []string) (Experiment, error) {
	experiment := Experiment{FormatVersion: FormatVersion, Observations: []Observation{}, Comparisons: []Comparison{}}
	if observer == nil || len(runIDs) < 2 || len(runIDs) > MaxObservations {
		return experiment, &Error{"INVALID_ARGUMENT", "experiment requires 2..20 observations"}
	}
	seen := map[string]bool{}
	for _, runID := range runIDs {
		if seen[runID] {
			return experiment, &Error{"INVALID_ARGUMENT", "duplicate source execution"}
		}
		seen[runID] = true
		if ctx.Err() != nil {
			return experiment, &Error{"CANCELLED", "experiment"}
		}
		observation, err := observer.Observe(ctx, runID)
		if err != nil {
			return experiment, err
		}
		experiment.Observations = append(experiment.Observations, observation)
	}
	baseline := experiment.Observations[0]
	for _, observation := range experiment.Observations[1:] {
		comparison, err := Compare(baseline, observation)
		if err != nil {
			return experiment, err
		}
		experiment.Comparisons = append(experiment.Comparisons, comparison)
	}
	experiment.Aggregate = aggregate(experiment)
	return experiment, nil
}

func aggregate(experiment Experiment) Aggregate {
	result := Aggregate{Observations: len(experiment.Observations), Comparisons: len(experiment.Comparisons)}
	for _, comparison := range experiment.Comparisons {
		switch comparison.Status {
		case Exact:
			result.Exact++
		case Partial:
			result.Partial++
		case Divergent:
			result.Divergent++
		case Insufficient:
			result.Insufficient++
		case Unsupported:
			result.Unsupported++
		}
		result.MeanEventAgreement += comparison.Metrics.EventAgreement
		result.MeanDimensionAgreement += comparison.Metrics.DimensionAgreement
	}
	if result.Comparisons > 0 {
		result.MeanEventAgreement /= float64(result.Comparisons)
		result.MeanDimensionAgreement /= float64(result.Comparisons)
	}
	return result
}
