package ipc

import (
	"context"
	"errors"

	"chronosync.local/engine/internal/analysis"
)

func analysisDispatch(req Request, service *analysis.Service) (any, error) {
	ctx := context.Background()
	switch req.Method {
	case "analysis.compare":
		var params struct {
			RunIDs []string        `json:"run_ids"`
			Policy analysis.Policy `json:"policy"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid analysis comparison")
		}
		return service.Compare(ctx, params.RunIDs, params.Policy)
	case "analysis.report.list":
		var params struct{}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid analysis report list")
		}
		return service.List(ctx)
	case "analysis.report.get":
		var params struct {
			ID string `json:"id"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid analysis report identity")
		}
		return service.Get(ctx, params.ID)
	case "analysis.report.reevaluate":
		var params struct {
			ID     string          `json:"id"`
			Policy analysis.Policy `json:"policy"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid analysis reevaluation")
		}
		return service.Reevaluate(ctx, params.ID, params.Policy)
	default:
		return nil, errors.New("unsupported analysis method")
	}
}
