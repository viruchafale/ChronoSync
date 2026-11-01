package ipc

import (
	"context"
	"errors"

	"chronosync.local/engine/internal/analysis"
	"chronosync.local/engine/internal/governance"
)

func governanceDispatch(req Request, service *governance.Service) (any, error) {
	ctx := context.Background()
	switch req.Method {
	case "benchmark.create":
		var params governance.CreateDataset
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid benchmark creation")
		}
		return service.CreateDataset(ctx, params)
	case "benchmark.import":
		var params struct {
			Bytes string `json:"bytes"`
		}
		if strictRequest(req.Params, &params) != nil || len(params.Bytes) == 0 || len(params.Bytes) > governance.MaxDatasetBytes {
			return nil, errors.New("invalid benchmark import")
		}
		return service.ImportDataset(ctx, []byte(params.Bytes))
	case "benchmark.list":
		var params struct{}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid benchmark list")
		}
		return service.ListDatasets(ctx)
	case "benchmark.get":
		var params struct {
			ID string `json:"id"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid benchmark identity")
		}
		return service.GetDataset(ctx, params.ID)
	case "benchmark.export":
		var params struct {
			ID      string `json:"id"`
			Consent bool   `json:"consent"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid benchmark export")
		}
		raw, record, err := service.ExportDataset(ctx, params.ID, params.Consent)
		if err != nil {
			return nil, err
		}
		return struct {
			Bytes  string                   `json:"bytes"`
			Record governance.DatasetRecord `json:"record"`
		}{string(raw), record}, nil
	case "benchmark.compare":
		var params struct {
			BaselineID  string `json:"baseline_id"`
			CandidateID string `json:"candidate_id"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid benchmark comparison")
		}
		return service.CompareDatasetVersions(ctx, params.BaselineID, params.CandidateID)
	case "benchmark.evaluate":
		var params struct {
			ID              string          `json:"id"`
			Policy          analysis.Policy `json:"policy"`
			ParentLineageID string          `json:"parent_lineage_id"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid benchmark evaluation")
		}
		return service.EvaluateDataset(ctx, params.ID, params.Policy, params.ParentLineageID)
	case "lineage.list":
		var params struct{}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid lineage list")
		}
		return service.ListLineages(ctx)
	case "lineage.get":
		var params struct {
			ID string `json:"id"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid lineage identity")
		}
		return service.GetLineage(ctx, params.ID)
	case "retention.set":
		var params struct {
			Kind   governance.ObjectKind      `json:"kind"`
			ID     string                     `json:"id"`
			State  governance.State           `json:"state"`
			Policy governance.RetentionPolicy `json:"policy"`
			Reason string                     `json:"reason"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid retention transition")
		}
		return service.SetRetention(ctx, params.Kind, params.ID, params.State, params.Policy, params.Reason)
	case "governance.audit":
		var params struct{}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid governance audit")
		}
		return service.Audit(ctx)
	case "governance.delete":
		var params struct {
			Kind    governance.ObjectKind `json:"kind"`
			ID      string                `json:"id"`
			Consent bool                  `json:"consent"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid governed deletion")
		}
		if err := service.Delete(ctx, params.Kind, params.ID, params.Consent); err != nil {
			return nil, err
		}
		return map[string]bool{"deleted": true}, nil
	default:
		return nil, errors.New("unsupported governance method")
	}
}
