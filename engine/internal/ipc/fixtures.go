package ipc

import (
	"context"
	"errors"

	"chronosync.local/engine/internal/fixtures"
	"chronosync.local/engine/internal/replay/unified"
)

func fixtureDispatch(req Request, service *fixtures.Service) (any, error) {
	ctx := context.Background()
	switch req.Method {
	case "fixture.eligibility", "fixture.create":
		var params struct {
			RunID string `json:"run_id"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid fixture run")
		}
		if req.Method == "fixture.eligibility" {
			return service.Eligibility(ctx, params.RunID)
		}
		return service.CreateFromRun(ctx, params.RunID)
	case "fixture.import":
		var params struct {
			Bytes string `json:"bytes"`
		}
		if strictRequest(req.Params, &params) != nil || len(params.Bytes) == 0 || len(params.Bytes) > fixtures.MaxFixtureBytes {
			return nil, errors.New("invalid fixture import")
		}
		return service.Import(ctx, []byte(params.Bytes))
	case "fixture.list":
		var params struct{}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid fixture list")
		}
		return service.List(ctx)
	case "fixture.inspect":
		var params struct {
			ID string `json:"id"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid fixture identity")
		}
		return service.Inspect(ctx, params.ID)
	case "fixture.export":
		var params struct {
			ID      string `json:"id"`
			Consent bool   `json:"consent"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid fixture export")
		}
		raw, record, err := service.Export(ctx, params.ID, params.Consent)
		if err != nil {
			return nil, err
		}
		return struct {
			Bytes  string          `json:"bytes"`
			Record fixtures.Record `json:"record"`
		}{string(raw), record}, nil
	case "fixture.delete":
		var params struct {
			ID      string `json:"id"`
			Consent bool   `json:"consent"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid fixture deletion")
		}
		if err := service.Delete(ctx, params.ID, params.Consent); err != nil {
			return nil, err
		}
		return map[string]bool{"deleted": true}, nil
	case "replay.evaluate":
		var params struct {
			RegressionID string       `json:"regression_id"`
			TransportID  string       `json:"transport_id"`
			Plan         unified.Plan `json:"plan"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid replay evaluation")
		}
		regressionRaw, regressionRecord, err := service.ReadForEvaluation(ctx, params.RegressionID)
		if err != nil {
			return nil, err
		}
		transportRaw, transportRecord, err := service.ReadForEvaluation(ctx, params.TransportID)
		if err != nil {
			return nil, err
		}
		if regressionRecord.Kind != fixtures.RegressionV1 || transportRecord.Kind != fixtures.TransportV1 {
			return nil, errors.New("fixture kinds do not match evaluation roles")
		}
		return unified.Evaluate(ctx, regressionRaw, transportRaw, params.Plan)
	default:
		return nil, errors.New("unsupported fixture method")
	}
}
