package ipc

import (
	"context"
	"encoding/base64"
	"errors"

	"chronosync.local/engine/internal/exchange"
)

func exchangeDispatch(req Request, service *exchange.Service) (any, error) {
	ctx := context.Background()
	switch req.Method {
	case "identity.register_os":
		var params struct {
			Label      string `json:"label"`
			Supersedes string `json:"supersedes"`
			PublicKey  string `json:"public_key"`
			Proof      string `json:"proof"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid OS-backed identity registration")
		}
		return service.RegisterOSIdentity(ctx, params.Label, params.Supersedes, params.PublicKey, params.Proof)
	case "identity.create":
		var params struct {
			Label      string `json:"label"`
			Supersedes string `json:"supersedes"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid identity creation")
		}
		return service.CreateIdentity(ctx, params.Label, params.Supersedes)
	case "identity.list":
		var params struct{}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid identity list")
		}
		return service.ListIdentities(ctx)
	case "identity.trust":
		var params struct {
			KeyID  string              `json:"key_id"`
			State  exchange.TrustState `json:"state"`
			Reason string              `json:"reason"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid trust change")
		}
		return service.SetTrust(ctx, params.KeyID, params.State, params.Reason)
	case "identity.migration.list":
		var params struct{}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid migration list")
		}
		return service.ListMigrations(ctx)
	case "identity.migration.begin":
		var params struct {
			KeyID string `json:"key_id"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid migration begin")
		}
		return service.BeginMigration(ctx, params.KeyID)
	case "identity.migration.verify":
		var params struct {
			KeyID     string `json:"key_id"`
			PublicKey string `json:"public_key"`
			Signature string `json:"signature"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid migration verification")
		}
		return service.VerifyMigration(ctx, params.KeyID, params.PublicKey, params.Signature)
	case "identity.migration.retire":
		var params struct {
			KeyID string `json:"key_id"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid migration retirement")
		}
		return service.RetireLegacy(ctx, params.KeyID)
	case "identity.migration.fail":
		var params struct {
			KeyID            string `json:"key_id"`
			Reason           string `json:"reason"`
			RecoveryRequired bool   `json:"recovery_required"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid migration failure")
		}
		return service.RecordMigrationFailure(ctx, params.KeyID, params.Reason, params.RecoveryRequired)
	case "identity.trust.audit":
		var params struct{}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid trust audit")
		}
		return service.AuditTrust(ctx), nil
	case "exchange.export":
		var params exchange.ExportRequest
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid exchange export")
		}
		return service.Export(ctx, params)
	case "exchange.export.begin":
		var params exchange.ExportRequest
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid exchange export begin")
		}
		return service.BeginExport(ctx, params)
	case "exchange.export.finish":
		var params struct {
			SessionID string `json:"session_id"`
			Signature string `json:"signature"`
		}
		if strictRequest(req.Params, &params) != nil {
			return nil, errors.New("invalid exchange export finish")
		}
		return service.FinishExport(ctx, params.SessionID, params.Signature)
	case "exchange.inspect", "exchange.import":
		var params struct {
			BytesBase64    string `json:"bytes_base64"`
			AllowUntrusted bool   `json:"allow_untrusted,omitempty"`
		}
		if strictRequest(req.Params, &params) != nil || len(params.BytesBase64) == 0 || len(params.BytesBase64) > base64.StdEncoding.EncodedLen(exchange.MaxPackageBytes) {
			return nil, errors.New("invalid exchange package")
		}
		raw, err := base64.StdEncoding.Strict().DecodeString(params.BytesBase64)
		if err != nil || len(raw) == 0 || len(raw) > exchange.MaxPackageBytes {
			return nil, errors.New("invalid exchange package encoding")
		}
		if req.Method == "exchange.inspect" {
			return service.Inspect(ctx, raw)
		}
		return service.Import(ctx, raw, params.AllowUntrusted)
	default:
		return nil, errors.New("unsupported exchange method")
	}
}
