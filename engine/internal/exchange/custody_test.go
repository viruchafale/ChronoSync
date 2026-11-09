package exchange

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOSCustodyRegistrationAndOneUseSigning(t *testing.T) {
	ctx := context.Background()
	reportRaw, report, comparison := testReport(t)
	datasetRaw, datasetID, _ := testDataset(t, report, comparison, false)
	service := newTestService(t, map[string][]byte{report.ReportID: reportRaw}, map[string][]byte{datasetID: datasetRaw})
	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index + 1)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	proof := ed25519.Sign(privateKey, registrationPayload(publicKey))
	record, err := service.RegisterOSIdentity(ctx, "Keychain signer", "", base64.StdEncoding.EncodeToString(publicKey), base64.StdEncoding.EncodeToString(proof))
	if err != nil || record.CustodyBackend != MacOSKeychain || record.CustodyStatus != CustodyUnchecked || record.Trust != Trusted {
		t.Fatal(record, err)
	}
	begin, err := service.BeginExport(ctx, ExportRequest{DatasetContentID: datasetID, IdentityKeyID: record.Identity.KeyID, IncludeReportIDs: []string{report.ReportID}})
	if err != nil || begin.Mode != "native_sign" || begin.SessionID == "" || begin.KeyID != record.Identity.KeyID {
		t.Fatal(begin, err)
	}
	payload, _ := base64.StdEncoding.DecodeString(begin.PayloadBase64)
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	exported, err := service.FinishExport(ctx, begin.SessionID, signature)
	if err != nil || exported.Verification.Status != ValidTrusted {
		t.Fatal(exported.Verification, err)
	}
	if _, err = service.FinishExport(ctx, begin.SessionID, signature); !exchangeError(err, "SIGNING_SESSION_EXPIRED") {
		t.Fatal("signing session was not one use", err)
	}
	begin, err = service.BeginExport(ctx, ExportRequest{DatasetContentID: datasetID, IdentityKeyID: record.Identity.KeyID, IncludeReportIDs: []string{report.ReportID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.FinishExport(ctx, begin.SessionID, base64.StdEncoding.EncodeToString(make([]byte, 64))); !exchangeError(err, "INVALID_SIGNATURE") {
		t.Fatal("malformed signing result accepted", err)
	}
}

func TestLegacyMigrationRestartIdempotencyAndCompatibility(t *testing.T) {
	ctx := context.Background()
	root := privateRoot(t)
	reportRaw, report, comparison := testReport(t)
	datasetRaw, datasetID, _ := testDataset(t, report, comparison, false)
	reports := &memoryReports{values: map[string][]byte{report.ReportID: reportRaw}}
	benchmarks := &memoryBenchmarks{values: map[string][]byte{datasetID: datasetRaw}}
	service := newTestServiceWithRoot(t, root, reports, benchmarks)
	legacy, err := service.CreateIdentity(ctx, "Legacy signer", "")
	if err != nil {
		t.Fatal(err)
	}
	oldPackage, err := service.Export(ctx, ExportRequest{DatasetContentID: datasetID, IdentityKeyID: legacy.Identity.KeyID, IncludeReportIDs: []string{report.ReportID}})
	if err != nil {
		t.Fatal(err)
	}
	begin, err := service.BeginMigration(ctx, legacy.Identity.KeyID)
	if err != nil || begin.Record.State != MigrationPending {
		t.Fatal(begin.Record, err)
	}
	seed, _ := base64.StdEncoding.DecodeString(begin.SeedBase64)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	challenge, _ := base64.StdEncoding.DecodeString(begin.ChallengeBase64)
	signature := ed25519.Sign(privateKey, migrationProofPayload(legacy.Identity.KeyID, challenge))
	verified, err := service.VerifyMigration(ctx, legacy.Identity.KeyID, base64.StdEncoding.EncodeToString(publicKey), base64.StdEncoding.EncodeToString(signature))
	if err != nil || verified.State != IdentityVerified {
		t.Fatal(verified, err)
	}
	service.Close()
	service, err = New(root, reports, benchmarks)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	resumed, err := service.BeginMigration(ctx, legacy.Identity.KeyID)
	if err != nil || resumed.Record.State != IdentityVerified || resumed.SeedBase64 != "" {
		t.Fatal(resumed, err)
	}
	completed, err := service.RetireLegacy(ctx, legacy.Identity.KeyID)
	if err != nil || completed.State != Migrated {
		t.Fatal(completed, err)
	}
	again, err := service.RetireLegacy(ctx, legacy.Identity.KeyID)
	if err != nil || again.State != Migrated {
		t.Fatal(again, err)
	}
	if _, err = os.Stat(filepath.Join(root, privateName(legacy.Identity.KeyID))); !os.IsNotExist(err) {
		t.Fatal("legacy seed was not retired", err)
	}
	raw, _ := base64.StdEncoding.DecodeString(oldPackage.BytesBase64)
	verification, err := service.Inspect(ctx, raw)
	if err != nil || verification.Status != ValidTrusted {
		t.Fatal(verification, err)
	}
	nativeBegin, err := service.BeginExport(ctx, ExportRequest{DatasetContentID: datasetID, IdentityKeyID: legacy.Identity.KeyID, IncludeReportIDs: []string{report.ReportID}})
	if err != nil || nativeBegin.Mode != "native_sign" {
		t.Fatal(nativeBegin, err)
	}
}

func TestMigrationMismatchAndTrustCorruptionRefuse(t *testing.T) {
	ctx := context.Background()
	root := privateRoot(t)
	service := newTestServiceWithRoot(t, root, &memoryReports{values: map[string][]byte{}}, &memoryBenchmarks{values: map[string][]byte{}})
	legacy, err := service.CreateIdentity(ctx, "Legacy signer", "")
	if err != nil {
		t.Fatal(err)
	}
	begin, err := service.BeginMigration(ctx, legacy.Identity.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	seed, _ := base64.StdEncoding.DecodeString(begin.SeedBase64)
	privateKey := ed25519.NewKeyFromSeed(seed)
	challenge, _ := base64.StdEncoding.DecodeString(begin.ChallengeBase64)
	badPublic := make([]byte, ed25519.PublicKeySize)
	signature := ed25519.Sign(privateKey, migrationProofPayload(legacy.Identity.KeyID, challenge))
	if _, err = service.VerifyMigration(ctx, legacy.Identity.KeyID, base64.StdEncoding.EncodeToString(badPublic), base64.StdEncoding.EncodeToString(signature)); !exchangeError(err, "MIGRATION_REFUSED") {
		t.Fatal("mismatched migration proof accepted", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "exchange/v1/trust"))
	if err != nil || len(entries) == 0 {
		t.Fatal(err)
	}
	path := filepath.Join(root, "exchange/v1/trust", entries[0].Name())
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)/2] ^= 1
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	audit := service.AuditTrust(ctx)
	if audit.Verified || len(audit.Issues) == 0 {
		t.Fatal("corrupt trust store reported verified", audit)
	}
}

func TestMigrationInterruptionConcurrencyAndIdentityCorruption(t *testing.T) {
	ctx := context.Background()
	root := privateRoot(t)
	reports := &memoryReports{values: map[string][]byte{}}
	benchmarks := &memoryBenchmarks{values: map[string][]byte{}}
	service := newTestServiceWithRoot(t, root, reports, benchmarks)
	legacy, err := service.CreateIdentity(ctx, "Interrupted legacy signer", "")
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.BeginMigration(ctx, legacy.Identity.KeyID)
	if err != nil || first.Record.State != MigrationPending {
		t.Fatal(first, err)
	}
	service.Close()
	service, err = New(root, reports, benchmarks)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	resumed, err := service.BeginMigration(ctx, legacy.Identity.KeyID)
	if err != nil || resumed.Record.State != MigrationPending || resumed.SeedBase64 == "" || resumed.ChallengeBase64 == first.ChallengeBase64 {
		t.Fatal("pending migration did not resume with a fresh proof challenge", resumed, err)
	}
	failed, err := service.RecordMigrationFailure(ctx, legacy.Identity.KeyID, "injected native denial", false)
	if err != nil || failed.State != MigrationFailed {
		t.Fatal(failed, err)
	}
	resumed, err = service.BeginMigration(ctx, legacy.Identity.KeyID)
	if err != nil || resumed.Record.State != MigrationFailed || resumed.SeedBase64 == "" {
		t.Fatal("failed migration was not safely resumable", resumed, err)
	}
	recovery, err := service.RecordMigrationFailure(ctx, legacy.Identity.KeyID, "injected ambiguous native result", true)
	if err != nil || recovery.State != RecoveryRequired {
		t.Fatal(recovery, err)
	}
	resumed, err = service.BeginMigration(ctx, legacy.Identity.KeyID)
	if err != nil || resumed.Record.State != RecoveryRequired || resumed.SeedBase64 == "" {
		t.Fatal("explicit recovery-required migration was not safely resumable", resumed, err)
	}
	if err = service.enter(); err != nil {
		t.Fatal(err)
	}
	if _, err = service.BeginMigration(ctx, legacy.Identity.KeyID); !exchangeError(err, "BUSY") {
		service.leave()
		t.Fatal("concurrent migration was not refused", err)
	}
	service.leave()
	publicPath := filepath.Join(root, publicName(legacy.Identity.KeyID))
	publicRaw, err := os.ReadFile(publicPath)
	if err != nil {
		t.Fatal(err)
	}
	publicRaw[len(publicRaw)/2] ^= 1
	if err = os.WriteFile(publicPath, publicRaw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = service.ListIdentities(ctx); err == nil {
		t.Fatal("corrupt identity was accepted")
	}
}

func TestMigrationRestartAfterKeychainCreated(t *testing.T) {
	ctx := context.Background()
	root := privateRoot(t)
	reports := &memoryReports{values: map[string][]byte{}}
	benchmarks := &memoryBenchmarks{values: map[string][]byte{}}
	service := newTestServiceWithRoot(t, root, reports, benchmarks)
	legacy, err := service.CreateIdentity(ctx, "Keychain-created signer", "")
	if err != nil {
		t.Fatal(err)
	}
	begin, err := service.BeginMigration(ctx, legacy.Identity.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	seed, _ := base64.StdEncoding.DecodeString(begin.SeedBase64)
	if _, err = service.appendMigration(ctx, legacy.Identity.KeyID, KeychainCreated, "injected interruption after matching Keychain creation"); err != nil {
		t.Fatal(err)
	}
	service.Close()
	service, err = New(root, reports, benchmarks)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	resumed, err := service.BeginMigration(ctx, legacy.Identity.KeyID)
	if err != nil || resumed.Record.State != KeychainCreated || resumed.SeedBase64 == "" {
		t.Fatal("keychain-created migration did not resume", resumed, err)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	challenge, _ := base64.StdEncoding.DecodeString(resumed.ChallengeBase64)
	signature := ed25519.Sign(privateKey, migrationProofPayload(legacy.Identity.KeyID, challenge))
	verified, err := service.VerifyMigration(ctx, legacy.Identity.KeyID, base64.StdEncoding.EncodeToString(publicKey), base64.StdEncoding.EncodeToString(signature))
	if err != nil || verified.State != IdentityVerified {
		t.Fatal(verified, err)
	}
}

func TestInvalidLegacySeedRefusesMigration(t *testing.T) {
	ctx := context.Background()
	root := privateRoot(t)
	service := newTestServiceWithRoot(t, root, &memoryReports{values: map[string][]byte{}}, &memoryBenchmarks{values: map[string][]byte{}})
	legacy, err := service.CreateIdentity(ctx, "Damaged legacy signer", "")
	if err != nil {
		t.Fatal(err)
	}
	privatePath := filepath.Join(root, privateName(legacy.Identity.KeyID))
	raw, err := os.ReadFile(privatePath)
	if err != nil {
		t.Fatal(err)
	}
	var record privateIdentity
	if err = json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	record.Seed = base64.StdEncoding.EncodeToString(make([]byte, 7))
	raw, _ = json.Marshal(record)
	if err = os.WriteFile(privatePath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = service.BeginMigration(ctx, legacy.Identity.KeyID); !exchangeError(err, "MIGRATION_REFUSED") {
		t.Fatal("invalid legacy seed was not refused", err)
	}
}
