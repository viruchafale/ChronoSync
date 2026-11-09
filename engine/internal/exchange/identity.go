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

func (s *Service) CreateIdentity(ctx context.Context, label, supersedes string) (IdentityRecord, error) {
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
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return IdentityRecord{}, &Error{"CRYPTO_ERROR", "identity generation"}
	}
	keyID := keyIdentifier(publicKey)
	identity := Identity{FormatVersion: FormatVersion, KeyID: keyID, Algorithm: AlgorithmEd25519, PublicKey: base64.StdEncoding.EncodeToString(publicKey), Label: label, CreatedAt: model.Now(), Supersedes: supersedes}
	if err = validateIdentity(identity); err != nil {
		return IdentityRecord{}, err
	}
	private := privateIdentity{FormatVersion: FormatVersion, KeyID: keyID, Algorithm: AlgorithmEd25519, Seed: base64.StdEncoding.EncodeToString(privateKey.Seed())}
	privateRaw, _ := json.Marshal(private)
	publicRaw, _ := json.Marshal(identity)
	if err = s.publish("exchange/v1/identities/private", privateName(keyID), keyID, privateRaw); err != nil {
		return IdentityRecord{}, err
	}
	if err = s.publish("exchange/v1/identities/public", publicName(keyID), keyID, publicRaw); err != nil {
		return IdentityRecord{}, err
	}
	status, err := s.appendTrust(ctx, keyID, Trusted, "local signing identity created")
	if err != nil {
		return IdentityRecord{}, err
	}
	return s.identityRecord(identity, status), nil
}

func (s *Service) ListIdentities(ctx context.Context) ([]IdentityRecord, error) {
	entries, err := s.readDirectory("exchange/v1/identities/public", MaxIdentities+1)
	if err != nil {
		return nil, err
	}
	statuses, err := s.trustInventory(ctx)
	if err != nil {
		return nil, err
	}
	result := []IdentityRecord{}
	for _, entry := range entries {
		if ctx.Err() != nil {
			return nil, &Error{"CANCELLED", "identity inventory"}
		}
		name := entry.Name()
		keyID := strings.TrimSuffix(name, ".json")
		if !strings.HasSuffix(name, ".json") || !validDigest(keyID) {
			return nil, &Error{"CORRUPT_IDENTITY", "unexpected identity entry"}
		}
		identity, readErr := s.getIdentity(keyID)
		if readErr != nil {
			return nil, readErr
		}
		status := statuses[keyID]
		if status.KeyID == "" {
			status = trustStatus{KeyID: keyID, State: Untrusted}
		}
		result = append(result, s.identityRecord(identity, status))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Identity.KeyID < result[j].Identity.KeyID })
	return result, nil
}

func (s *Service) SetTrust(ctx context.Context, keyID string, state TrustState, reason string) (IdentityRecord, error) {
	if err := s.enter(); err != nil {
		return IdentityRecord{}, err
	}
	defer s.leave()
	identity, err := s.getIdentity(keyID)
	if err != nil {
		return IdentityRecord{}, err
	}
	status, err := s.appendTrust(ctx, keyID, state, reason)
	if err != nil {
		return IdentityRecord{}, err
	}
	return s.identityRecord(identity, status), nil
}

func (s *Service) getIdentity(keyID string) (Identity, error) {
	if !validDigest(keyID) {
		return Identity{}, &Error{"INVALID_IDENTITY", "key identity"}
	}
	raw, err := s.readRegular(publicName(keyID), 16*1024)
	if errors.Is(err, os.ErrNotExist) {
		return Identity{}, &Error{"NOT_FOUND", "signing identity"}
	}
	if err != nil {
		return Identity{}, err
	}
	var identity Identity
	if strictCanonical(raw, &identity) != nil || validateIdentity(identity) != nil || identity.KeyID != keyID {
		return Identity{}, &Error{"CORRUPT_IDENTITY", "public identity"}
	}
	return identity, nil
}

func (s *Service) observeIdentity(identity Identity) (Identity, error) {
	if err := validateIdentity(identity); err != nil {
		return Identity{}, err
	}
	if existing, err := s.getIdentity(identity.KeyID); err == nil {
		if existing.Algorithm != identity.Algorithm || existing.PublicKey != identity.PublicKey {
			return Identity{}, &Error{"IDENTITY_CONFLICT", "key identity has different public key"}
		}
		return existing, nil
	}
	raw, _ := json.Marshal(identity)
	if err := s.publish("exchange/v1/identities/public", publicName(identity.KeyID), identity.KeyID, raw); err != nil {
		return Identity{}, err
	}
	return identity, nil
}

func (s *Service) privateKey(keyID string) (ed25519.PrivateKey, error) {
	raw, err := s.readRegular(privateName(keyID), 4*1024)
	if errors.Is(err, os.ErrNotExist) {
		return nil, &Error{"PRIVATE_KEY_UNAVAILABLE", "signing identity"}
	}
	if err != nil {
		return nil, err
	}
	var record privateIdentity
	if strictCanonical(raw, &record) != nil || record.FormatVersion != FormatVersion || record.KeyID != keyID || record.Algorithm != AlgorithmEd25519 {
		return nil, &Error{"CORRUPT_IDENTITY", "private identity"}
	}
	seed, decodeErr := base64.StdEncoding.Strict().DecodeString(record.Seed)
	if decodeErr != nil || len(seed) != ed25519.SeedSize {
		return nil, &Error{"CORRUPT_IDENTITY", "private seed"}
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	if keyIdentifier(privateKey.Public().(ed25519.PublicKey)) != keyID {
		return nil, &Error{"CORRUPT_IDENTITY", "private/public mismatch"}
	}
	return privateKey, nil
}

func validateIdentity(identity Identity) error {
	if identity.FormatVersion != FormatVersion || identity.Algorithm != AlgorithmEd25519 || !validDigest(identity.KeyID) || !validLabel(identity.Label) || identity.Supersedes == identity.KeyID || identity.Supersedes != "" && !validDigest(identity.Supersedes) {
		return &Error{"INVALID_IDENTITY", "identity contract"}
	}
	created, err := time.Parse(time.RFC3339Nano, identity.CreatedAt)
	if err != nil || created.Location() != time.UTC {
		return &Error{"INVALID_IDENTITY", "identity timestamp"}
	}
	publicKey, err := base64.StdEncoding.Strict().DecodeString(identity.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize || keyIdentifier(publicKey) != identity.KeyID {
		return &Error{"INVALID_IDENTITY", "public key or key identity"}
	}
	return nil
}

func keyIdentifier(publicKey []byte) string { return digest([]byte(keyDomain), publicKey) }
func validLabel(value string) bool {
	return len(value) > 0 && len(value) <= MaxIdentityLabel && !strings.ContainsRune(value, 0)
}

func (s *Service) hasPrivateKey(keyID string) bool {
	info, err := s.root.Lstat(privateName(keyID))
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0077 == 0
}

func (s *Service) identityRecord(identity Identity, status trustStatus) IdentityRecord {
	backend, custodyStatus := s.custodyState(identity.KeyID)
	warnings := []string{"A valid trusted signature does not establish package safety, benchmark quality, or execution authorization."}
	if backend == LegacyFile {
		warnings = append(warnings, "Legacy seed uses ordinary local file protection and is eligible for explicit Keychain migration.")
	} else if backend == MacOSKeychain {
		warnings = append(warnings, "macOS Keychain custody is OS-backed but not Secure Enclave, hardware-backed, or resistant to trusted-process compromise.")
	}
	migrationState := MigrationState("")
	if states, err := s.migrationInventory(context.Background()); err == nil {
		migrationState = states[identity.KeyID].State
	} else {
		custodyStatus = CustodyRecoveryRequired
		warnings = append(warnings, "Migration history failed validation; custody state requires recovery review.")
	}
	return IdentityRecord{Identity: identity, Trust: status.State, HasPrivateKey: backend == LegacyFile || backend == MacOSKeychain, TrustRevision: status.Revision, TrustUpdatedAt: status.UpdatedAt, Warnings: warnings, CustodyBackend: backend, CustodyStatus: custodyStatus, MigrationState: migrationState}
}

type trustStatus struct {
	KeyID     string
	State     TrustState
	Revision  uint64
	UpdatedAt string
}

func (s *Service) appendTrust(ctx context.Context, keyID string, state TrustState, reason string) (trustStatus, error) {
	if !validDigest(keyID) || len(reason) < 1 || len(reason) > 256 || strings.ContainsRune(reason, 0) {
		return trustStatus{}, &Error{"INVALID_TRUST", "identity or reason"}
	}
	statuses, err := s.trustInventory(ctx)
	if err != nil {
		return trustStatus{}, err
	}
	current := statuses[keyID]
	if current.KeyID == "" {
		current = trustStatus{KeyID: keyID, State: Untrusted}
	}
	if !allowedTrustTransition(current.State, state) {
		return trustStatus{}, &Error{"INVALID_TRUST", "trust transition"}
	}
	event := TrustEvent{FormatVersion: FormatVersion, EventID: model.ID(), KeyID: keyID, Revision: current.Revision + 1, PreviousState: current.State, State: state, Reason: reason, CreatedAt: model.Now()}
	unsigned, _ := json.Marshal(event)
	event.SHA256 = digest([]byte(trustIntegrityDomain), unsigned)
	raw, _ := json.Marshal(event)
	if err = s.publish("exchange/v1/trust", trustName(event.EventID), event.EventID, raw); err != nil {
		return trustStatus{}, err
	}
	return trustStatus{KeyID: keyID, State: state, Revision: event.Revision, UpdatedAt: event.CreatedAt}, nil
}

func (s *Service) trustInventory(ctx context.Context) (map[string]trustStatus, error) {
	entries, err := s.readDirectory("exchange/v1/trust", MaxTrustEvents+1)
	if err != nil {
		return nil, err
	}
	events := []TrustEvent{}
	for _, entry := range entries {
		if ctx.Err() != nil {
			return nil, &Error{"CANCELLED", "trust inventory"}
		}
		name := entry.Name()
		id := strings.TrimSuffix(name, ".json")
		if !strings.HasSuffix(name, ".json") || !validModelID(id) {
			return nil, &Error{"CORRUPT_TRUST", "unexpected trust entry"}
		}
		raw, readErr := s.readRegular("exchange/v1/trust/"+name, 64*1024)
		if readErr != nil {
			return nil, readErr
		}
		var event TrustEvent
		if strictCanonical(raw, &event) != nil || event.FormatVersion != FormatVersion || event.EventID != id || !validDigest(event.KeyID) || !validUTC(event.CreatedAt) || len(event.Reason) < 1 || len(event.Reason) > 256 || strings.ContainsRune(event.Reason, 0) {
			return nil, &Error{"CORRUPT_TRUST", "trust event contract"}
		}
		if _, identityErr := s.getIdentity(event.KeyID); identityErr != nil {
			return nil, &Error{"CORRUPT_TRUST", "trust event identity"}
		}
		digestValue := event.SHA256
		event.SHA256 = ""
		unsigned, _ := json.Marshal(event)
		event.SHA256 = digestValue
		if digestValue != digest([]byte(trustIntegrityDomain), unsigned) {
			return nil, &Error{"CORRUPT_TRUST", "trust event integrity"}
		}
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].KeyID == events[j].KeyID {
			return events[i].Revision < events[j].Revision
		}
		return events[i].KeyID < events[j].KeyID
	})
	statuses := map[string]trustStatus{}
	for _, event := range events {
		current := statuses[event.KeyID]
		if current.KeyID == "" {
			current = trustStatus{KeyID: event.KeyID, State: Untrusted}
		}
		if event.Revision != current.Revision+1 || event.PreviousState != current.State || !allowedTrustTransition(current.State, event.State) {
			return nil, &Error{"CORRUPT_TRUST", "trust history"}
		}
		statuses[event.KeyID] = trustStatus{KeyID: event.KeyID, State: event.State, Revision: event.Revision, UpdatedAt: event.CreatedAt}
	}
	return statuses, nil
}

func allowedTrustTransition(from, to TrustState) bool {
	if from == Revoked {
		return false
	}
	if from == Untrusted {
		return to == Trusted || to == Revoked
	}
	return from == Trusted && (to == Untrusted || to == Revoked)
}

func (s *Service) trustState(ctx context.Context, keyID string) (trustStatus, error) {
	statuses, err := s.trustInventory(ctx)
	if err != nil {
		return trustStatus{}, err
	}
	if status, ok := statuses[keyID]; ok {
		return status, nil
	}
	return trustStatus{KeyID: keyID, State: Untrusted}, nil
}

func publicName(id string) string  { return "exchange/v1/identities/public/" + id + ".json" }
func privateName(id string) string { return "exchange/v1/identities/private/" + id + ".json" }
func trustName(id string) string   { return "exchange/v1/trust/" + id + ".json" }
func validModelID(id string) bool  { return len(id) == 32 && model.ValidID(id) }
