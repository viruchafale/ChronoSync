package exchange

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"time"

	"chronosync.local/engine/internal/model"
)

func registrationPayload(publicKey []byte) []byte {
	return append(append([]byte(nil), []byte(registrationDomain)...), publicKey...)
}

func migrationProofPayload(keyID string, challenge []byte) []byte {
	result := append([]byte(nil), []byte(migrationProofDomain)...)
	result = append(result, []byte(keyID)...)
	result = append(result, 0)
	return append(result, challenge...)
}

func (s *Service) RegisterOSIdentity(ctx context.Context, label, supersedes, publicBase64, proofBase64 string) (IdentityRecord, error) {
	if err := s.enter(); err != nil {
		return IdentityRecord{}, err
	}
	defer s.leave()
	if ctx.Err() != nil || !validLabel(label) {
		return IdentityRecord{}, &Error{"INVALID_IDENTITY", "label or context"}
	}
	if supersedes != "" {
		if _, err := s.getIdentity(supersedes); err != nil {
			return IdentityRecord{}, &Error{"INVALID_IDENTITY", "superseded identity"}
		}
	}
	publicKey, err := base64.StdEncoding.Strict().DecodeString(publicBase64)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return IdentityRecord{}, &Error{"INVALID_IDENTITY", "OS custody public key"}
	}
	proof, err := base64.StdEncoding.Strict().DecodeString(proofBase64)
	if err != nil || len(proof) != ed25519.SignatureSize || !ed25519.Verify(publicKey, registrationPayload(publicKey), proof) {
		return IdentityRecord{}, &Error{"INVALID_SIGNATURE", "OS custody registration proof"}
	}
	keyID := keyIdentifier(publicKey)
	if _, err = s.getIdentity(keyID); err == nil {
		return IdentityRecord{}, &Error{"IDENTITY_CONFLICT", "identity already exists"}
	}
	identity := Identity{FormatVersion: FormatVersion, KeyID: keyID, Algorithm: AlgorithmEd25519, PublicKey: publicBase64, Label: label, CreatedAt: model.Now(), Supersedes: supersedes}
	publicRaw, _ := json.Marshal(identity)
	if err = s.publish("exchange/v1/identities/public", publicName(keyID), keyID, publicRaw); err != nil {
		return IdentityRecord{}, err
	}
	if err = s.publishCustody(identity.KeyID, MacOSKeychain); err != nil {
		return IdentityRecord{}, err
	}
	status, err := s.appendTrust(ctx, keyID, Trusted, "OS-backed signing identity created with native confirmation")
	if err != nil {
		return IdentityRecord{}, err
	}
	return s.identityRecord(identity, status), nil
}

func (s *Service) publishCustody(keyID string, backend CustodyBackend) error {
	if !validDigest(keyID) || backend != MacOSKeychain {
		return &Error{"INVALID_CUSTODY", "custody record"}
	}
	record := CustodyRecord{FormatVersion: FormatVersion, KeyID: keyID, Backend: backend, CreatedAt: model.Now()}
	raw, _ := json.Marshal(record)
	if existing, err := s.readCustody(keyID); err == nil {
		if existing.Backend == backend {
			return nil
		}
		return &Error{"CUSTODY_CONFLICT", "identity custody backend"}
	}
	return s.publish("exchange/v1/custody", custodyName(keyID), keyID, raw)
}

func (s *Service) readCustody(keyID string) (CustodyRecord, error) {
	raw, err := s.readRegular(custodyName(keyID), 8*1024)
	if err != nil {
		return CustodyRecord{}, err
	}
	var record CustodyRecord
	if strictCanonical(raw, &record) != nil || record.FormatVersion != FormatVersion || record.KeyID != keyID || record.Backend != MacOSKeychain || !validUTC(record.CreatedAt) {
		return CustodyRecord{}, &Error{"CORRUPT_CUSTODY", "custody record"}
	}
	return record, nil
}

func (s *Service) custodyState(keyID string) (CustodyBackend, CustodyStatus) {
	if _, err := s.readCustody(keyID); err == nil {
		return MacOSKeychain, CustodyUnchecked
	}
	if s.hasPrivateKey(keyID) {
		return LegacyFile, CustodyAvailable
	}
	return NoCustody, CustodyMissing
}

func (s *Service) ListMigrations(ctx context.Context) ([]MigrationRecord, error) {
	identities, err := s.ListIdentities(ctx)
	if err != nil {
		return nil, err
	}
	states, err := s.migrationInventory(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]MigrationRecord, 0, len(identities))
	for _, identity := range identities {
		record := states[identity.Identity.KeyID]
		if record.KeyID == "" && identity.CustodyBackend == LegacyFile {
			record = MigrationRecord{KeyID: identity.Identity.KeyID, State: LegacyDetected, Eligible: true, CustodyBackend: LegacyFile, Detail: "Validated Phase 8 seed is eligible for explicit migration."}
		}
		if record.KeyID != "" {
			result = append(result, record)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].KeyID < result[j].KeyID })
	return result, nil
}

func (s *Service) BeginMigration(ctx context.Context, keyID string) (MigrationBeginResult, error) {
	if err := s.enter(); err != nil {
		return MigrationBeginResult{}, err
	}
	defer s.leave()
	identity, err := s.getIdentity(keyID)
	if err != nil {
		return MigrationBeginResult{}, err
	}
	states, err := s.migrationInventory(ctx)
	if err != nil {
		return MigrationBeginResult{}, err
	}
	current := states[keyID]
	if current.State == Migrated {
		return MigrationBeginResult{Record: current}, nil
	}
	if current.State == IdentityVerified || current.State == LegacyRetirementPending {
		return MigrationBeginResult{Record: current}, nil
	}
	privateKey, err := s.privateKey(keyID)
	if err != nil || keyIdentifier(privateKey.Public().(ed25519.PublicKey)) != identity.KeyID {
		return MigrationBeginResult{}, &Error{"MIGRATION_REFUSED", "legacy private/public key mismatch or corruption"}
	}
	if current.KeyID == "" {
		current, err = s.appendMigration(ctx, keyID, MigrationPending, "native-confirmed migration started")
		if err != nil {
			return MigrationBeginResult{}, err
		}
	}
	if current.State != MigrationPending && current.State != MigrationFailed && current.State != KeychainCreated && current.State != RecoveryRequired {
		return MigrationBeginResult{}, &Error{"MIGRATION_CONFLICT", "migration state cannot begin"}
	}
	challenge := make([]byte, 32)
	if _, err = rand.Read(challenge); err != nil {
		return MigrationBeginResult{}, &Error{"CRYPTO_ERROR", "migration challenge"}
	}
	s.migrationChallenges[keyID] = challenge
	seed := privateKey.Seed()
	return MigrationBeginResult{Record: current, SeedBase64: base64.StdEncoding.EncodeToString(seed), ChallengeBase64: base64.StdEncoding.EncodeToString(challenge)}, nil
}

func (s *Service) VerifyMigration(ctx context.Context, keyID, publicBase64, signatureBase64 string) (MigrationRecord, error) {
	if err := s.enter(); err != nil {
		return MigrationRecord{}, err
	}
	defer s.leave()
	identity, err := s.getIdentity(keyID)
	if err != nil {
		return MigrationRecord{}, err
	}
	challenge, ok := s.migrationChallenges[keyID]
	delete(s.migrationChallenges, keyID)
	if !ok {
		return MigrationRecord{}, &Error{"MIGRATION_SESSION_EXPIRED", "migration proof challenge"}
	}
	publicKey, publicErr := base64.StdEncoding.Strict().DecodeString(publicBase64)
	signature, signatureErr := base64.StdEncoding.Strict().DecodeString(signatureBase64)
	if publicErr != nil || signatureErr != nil || publicBase64 != identity.PublicKey || keyIdentifier(publicKey) != keyID || len(signature) != ed25519.SignatureSize || !ed25519.Verify(publicKey, migrationProofPayload(keyID, challenge), signature) {
		return MigrationRecord{}, &Error{"MIGRATION_REFUSED", "Keychain public key or proof mismatch"}
	}
	states, err := s.migrationInventory(ctx)
	if err != nil {
		return MigrationRecord{}, err
	}
	if states[keyID].State != KeychainCreated {
		if _, err = s.appendMigration(ctx, keyID, KeychainCreated, "matching Keychain item created or resumed"); err != nil {
			return MigrationRecord{}, err
		}
	}
	if err = s.publishCustody(keyID, MacOSKeychain); err != nil {
		return MigrationRecord{}, err
	}
	return s.appendMigration(ctx, keyID, IdentityVerified, "Keychain public identity and proof signature verified")
}

func (s *Service) RetireLegacy(ctx context.Context, keyID string) (MigrationRecord, error) {
	if err := s.enter(); err != nil {
		return MigrationRecord{}, err
	}
	defer s.leave()
	states, err := s.migrationInventory(ctx)
	if err != nil {
		return MigrationRecord{}, err
	}
	current := states[keyID]
	if current.State == Migrated {
		return current, nil
	}
	if current.State != IdentityVerified && current.State != LegacyRetirementPending {
		return MigrationRecord{}, &Error{"MIGRATION_CONFLICT", "identity has not been verified"}
	}
	if current.State == IdentityVerified {
		if _, err = s.appendMigration(ctx, keyID, LegacyRetirementPending, "Keychain proof complete; legacy retirement pending"); err != nil {
			return MigrationRecord{}, err
		}
	}
	if err = s.root.Remove(privateName(keyID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return MigrationRecord{}, &Error{"STORAGE_ERROR", "legacy seed retirement"}
	}
	if err = s.sync("exchange/v1/identities/private"); err != nil {
		return MigrationRecord{}, err
	}
	return s.appendMigration(ctx, keyID, Migrated, "legacy seed retired after verified Keychain custody")
}

func (s *Service) RecordMigrationFailure(ctx context.Context, keyID, reason string, recoveryRequired bool) (MigrationRecord, error) {
	if err := s.enter(); err != nil {
		return MigrationRecord{}, err
	}
	defer s.leave()
	state := MigrationFailed
	if recoveryRequired {
		state = RecoveryRequired
	}
	return s.appendMigration(ctx, keyID, state, reason)
}

func (s *Service) appendMigration(ctx context.Context, keyID string, state MigrationState, reason string) (MigrationRecord, error) {
	if !validDigest(keyID) || len(reason) < 1 || len(reason) > 256 || strings.ContainsRune(reason, 0) {
		return MigrationRecord{}, &Error{"INVALID_MIGRATION", "event fields"}
	}
	states, err := s.migrationInventory(ctx)
	if err != nil {
		return MigrationRecord{}, err
	}
	current := states[keyID]
	if !allowedMigrationTransition(current.State, state) {
		return MigrationRecord{}, &Error{"INVALID_MIGRATION", "state transition"}
	}
	event := MigrationEvent{FormatVersion: FormatVersion, EventID: model.ID(), KeyID: keyID, Revision: current.Revision + 1, PreviousState: current.State, State: state, Reason: reason, CreatedAt: model.Now()}
	unsigned, _ := json.Marshal(event)
	event.SHA256 = digest([]byte(migrationDomain), unsigned)
	raw, _ := json.Marshal(event)
	if err = s.publish("exchange/v1/migration", migrationName(event.EventID), event.EventID, raw); err != nil {
		return MigrationRecord{}, err
	}
	return MigrationRecord{KeyID: keyID, State: state, Revision: event.Revision, UpdatedAt: event.CreatedAt, Eligible: state != Migrated, CustodyBackend: custodyForMigration(state), Detail: reason}, nil
}

func (s *Service) migrationInventory(ctx context.Context) (map[string]MigrationRecord, error) {
	entries, err := s.readDirectory("exchange/v1/migration", MaxMigrationEvents+1)
	if err != nil {
		return nil, err
	}
	events := make([]MigrationEvent, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		id := strings.TrimSuffix(name, ".json")
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !strings.HasSuffix(name, ".json") || !validModelID(id) {
			return nil, &Error{"CORRUPT_MIGRATION", "migration inventory"}
		}
		raw, readErr := s.readRegular("exchange/v1/migration/"+name, 64*1024)
		var event MigrationEvent
		if readErr != nil || strictCanonical(raw, &event) != nil || event.FormatVersion != FormatVersion || event.EventID != id || !validDigest(event.KeyID) || !validUTC(event.CreatedAt) || len(event.Reason) < 1 || len(event.Reason) > 256 {
			return nil, &Error{"CORRUPT_MIGRATION", "migration event"}
		}
		stored := event.SHA256
		event.SHA256 = ""
		unsigned, _ := json.Marshal(event)
		event.SHA256 = stored
		if stored != digest([]byte(migrationDomain), unsigned) {
			return nil, &Error{"CORRUPT_MIGRATION", "migration integrity"}
		}
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].KeyID == events[j].KeyID {
			return events[i].Revision < events[j].Revision
		}
		return events[i].KeyID < events[j].KeyID
	})
	states := map[string]MigrationRecord{}
	for _, event := range events {
		current := states[event.KeyID]
		if event.Revision != current.Revision+1 || event.PreviousState != current.State || !allowedMigrationTransition(current.State, event.State) {
			return nil, &Error{"CORRUPT_MIGRATION", "migration history"}
		}
		states[event.KeyID] = MigrationRecord{KeyID: event.KeyID, State: event.State, Revision: event.Revision, UpdatedAt: event.CreatedAt, Eligible: event.State != Migrated, CustodyBackend: custodyForMigration(event.State), Detail: event.Reason}
	}
	return states, nil
}

func allowedMigrationTransition(from, to MigrationState) bool {
	switch from {
	case "":
		return to == MigrationPending
	case MigrationPending, MigrationFailed, RecoveryRequired:
		return to == KeychainCreated || to == MigrationFailed || to == RecoveryRequired
	case KeychainCreated:
		return to == IdentityVerified || to == MigrationFailed || to == RecoveryRequired
	case IdentityVerified:
		return to == LegacyRetirementPending || to == RecoveryRequired
	case LegacyRetirementPending:
		return to == Migrated || to == RecoveryRequired
	default:
		return false
	}
}

func custodyForMigration(state MigrationState) CustodyBackend {
	if state == KeychainCreated || state == IdentityVerified || state == LegacyRetirementPending || state == Migrated {
		return MacOSKeychain
	}
	return LegacyFile
}

func (s *Service) AuditTrust(ctx context.Context) TrustAudit {
	audit := TrustAudit{FormatVersion: FormatVersion, Issues: []string{}}
	identities, err := s.ListIdentities(ctx)
	if err != nil {
		audit.Issues = append(audit.Issues, err.Error())
		return audit
	}
	audit.Identities = len(identities)
	statuses, err := s.trustInventory(ctx)
	if err != nil {
		audit.Issues = append(audit.Issues, err.Error())
		return audit
	}
	for _, status := range statuses {
		audit.Events += int(status.Revision)
		if status.State == Revoked {
			audit.Revoked++
		}
	}
	audit.Verified = true
	return audit
}

func validUTC(value string) bool {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return err == nil && parsed.Location() == time.UTC
}

func custodyName(id string) string   { return "exchange/v1/custody/" + id + ".json" }
func migrationName(id string) string { return "exchange/v1/migration/" + id + ".json" }
