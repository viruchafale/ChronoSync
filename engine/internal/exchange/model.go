// Package exchange implements non-executable signed benchmark exchange.
// It has no provider, command, agent, network, snapshot, admission or approval
// capability.
package exchange

import "chronosync.local/engine/internal/governance"

const (
	FormatVersion        = 1
	AlgorithmEd25519     = "ed25519"
	MaxPackageBytes      = 8 * 1024 * 1024
	MaxEntries           = 100
	MaxManifestBytes     = 128 * 1024
	MaxSignatureBytes    = 16 * 1024
	MaxIdentityLabel     = 128
	MaxTrustEvents       = 10_000
	MaxMigrationEvents   = 10_000
	MaxIdentities        = 1_000
	MaxExchangeStorage   = 64 * 1024 * 1024
	manifestPath         = "manifest.json"
	benchmarkPath        = "benchmark.benchmark"
	signaturePath        = "signature.json"
	packageDomain        = "chronosync-exchange-package-v1\x00"
	signatureDomain      = "chronosync-exchange-signature-v1\x00"
	keyDomain            = "chronosync-signing-key-v1\x00ed25519\x00"
	trustIntegrityDomain = "chronosync-trust-event-v1\x00"
	migrationDomain      = "chronosync-key-migration-event-v1\x00"
	registrationDomain   = "chronosync-key-custody-registration-v1\x00"
	migrationProofDomain = "chronosync-key-migration-proof-v1\x00"
)

type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Detail }

type TrustState string

const (
	Untrusted TrustState = "untrusted"
	Trusted   TrustState = "trusted"
	Revoked   TrustState = "revoked"
)

type Identity struct {
	FormatVersion int    `json:"format_version"`
	KeyID         string `json:"key_id"`
	Algorithm     string `json:"algorithm"`
	PublicKey     string `json:"public_key"`
	Label         string `json:"label"`
	CreatedAt     string `json:"created_at"`
	Supersedes    string `json:"supersedes,omitempty"`
}

type IdentityRecord struct {
	Identity       Identity       `json:"identity"`
	Trust          TrustState     `json:"trust"`
	HasPrivateKey  bool           `json:"has_private_key"`
	TrustRevision  uint64         `json:"trust_revision"`
	TrustUpdatedAt string         `json:"trust_updated_at,omitempty"`
	Warnings       []string       `json:"warnings"`
	CustodyBackend CustodyBackend `json:"custody_backend"`
	CustodyStatus  CustodyStatus  `json:"custody_status"`
	MigrationState MigrationState `json:"migration_state,omitempty"`
}

type CustodyBackend string

const (
	NoCustody          CustodyBackend = "none"
	LegacyFile         CustodyBackend = "legacy_file"
	MacOSKeychain      CustodyBackend = "macos_keychain"
	UnsupportedCustody CustodyBackend = "unsupported"
)

type CustodyStatus string

const (
	CustodyUnchecked        CustodyStatus = "unchecked"
	CustodyAvailable        CustodyStatus = "available"
	CustodyDenied           CustodyStatus = "denied"
	CustodyMissing          CustodyStatus = "missing"
	CustodyCorrupt          CustodyStatus = "corrupt"
	CustodyUnsupported      CustodyStatus = "unsupported"
	CustodyRecoveryRequired CustodyStatus = "recovery_required"
)

type MigrationState string

const (
	LegacyDetected          MigrationState = "legacy_detected"
	MigrationPending        MigrationState = "migration_pending"
	KeychainCreated         MigrationState = "keychain_created"
	IdentityVerified        MigrationState = "identity_verified"
	LegacyRetirementPending MigrationState = "legacy_retirement_pending"
	Migrated                MigrationState = "migrated"
	MigrationFailed         MigrationState = "migration_failed"
	RecoveryRequired        MigrationState = "recovery_required"
)

type CustodyRecord struct {
	FormatVersion int            `json:"format_version"`
	KeyID         string         `json:"key_id"`
	Backend       CustodyBackend `json:"backend"`
	CreatedAt     string         `json:"created_at"`
}

type MigrationEvent struct {
	FormatVersion int            `json:"format_version"`
	EventID       string         `json:"event_id"`
	KeyID         string         `json:"key_id"`
	Revision      uint64         `json:"revision"`
	PreviousState MigrationState `json:"previous_state,omitempty"`
	State         MigrationState `json:"state"`
	Reason        string         `json:"reason"`
	CreatedAt     string         `json:"created_at"`
	SHA256        string         `json:"sha256"`
}

type MigrationRecord struct {
	KeyID          string         `json:"key_id"`
	State          MigrationState `json:"state"`
	Revision       uint64         `json:"revision"`
	UpdatedAt      string         `json:"updated_at"`
	Eligible       bool           `json:"eligible"`
	CustodyBackend CustodyBackend `json:"custody_backend"`
	Detail         string         `json:"detail"`
}

type MigrationBeginResult struct {
	Record          MigrationRecord `json:"record"`
	SeedBase64      string          `json:"seed_base64,omitempty"`
	ChallengeBase64 string          `json:"challenge_base64,omitempty"`
}

type TrustAudit struct {
	FormatVersion int      `json:"format_version"`
	Verified      bool     `json:"verified"`
	Identities    int      `json:"identities"`
	Events        int      `json:"events"`
	Revoked       int      `json:"revoked"`
	Issues        []string `json:"issues"`
}

type ExportBeginResult struct {
	Mode          string        `json:"mode"`
	SessionID     string        `json:"session_id,omitempty"`
	KeyID         string        `json:"key_id,omitempty"`
	PayloadBase64 string        `json:"payload_base64,omitempty"`
	Completed     *ExportResult `json:"completed,omitempty"`
}

type privateIdentity struct {
	FormatVersion int    `json:"format_version"`
	KeyID         string `json:"key_id"`
	Algorithm     string `json:"algorithm"`
	Seed          string `json:"seed"`
}

type TrustEvent struct {
	FormatVersion int        `json:"format_version"`
	EventID       string     `json:"event_id"`
	KeyID         string     `json:"key_id"`
	Revision      uint64     `json:"revision"`
	PreviousState TrustState `json:"previous_state"`
	State         TrustState `json:"state"`
	Reason        string     `json:"reason"`
	CreatedAt     string     `json:"created_at"`
	SHA256        string     `json:"sha256"`
}

type Entry struct {
	Path   string `json:"path"`
	Role   string `json:"role"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type PrivacyManifest struct {
	RawCommands      bool     `json:"raw_commands"`
	RawOutputs       bool     `json:"raw_outputs"`
	WorkspacePaths   bool     `json:"workspace_paths"`
	Credentials      bool     `json:"credentials"`
	Environment      bool     `json:"environment"`
	PrivateKeys      bool     `json:"private_keys"`
	IncludedEvidence []string `json:"included_evidence"`
	OmittedEvidence  []string `json:"omitted_evidence"`
}

type Manifest struct {
	FormatVersion      int             `json:"format_version"`
	Domain             string          `json:"domain"`
	DatasetContentID   string          `json:"dataset_content_id"`
	BenchmarkID        string          `json:"benchmark_id"`
	BenchmarkVersion   uint64          `json:"benchmark_version"`
	CompatibilityModel string          `json:"compatibility_model"`
	Entries            []Entry         `json:"entries"`
	IncludedReportIDs  []string        `json:"included_report_ids"`
	OmittedReportIDs   []string        `json:"omitted_report_ids"`
	Privacy            PrivacyManifest `json:"privacy"`
}

type SignatureEnvelope struct {
	FormatVersion int      `json:"format_version"`
	Algorithm     string   `json:"algorithm"`
	Identity      Identity `json:"identity"`
	Signature     string   `json:"signature"`
}

type VerificationStatus string

const (
	ValidTrusted         VerificationStatus = "valid_trusted"
	ValidUntrusted       VerificationStatus = "valid_untrusted"
	InvalidSignature     VerificationStatus = "invalid_signature"
	RevokedIdentity      VerificationStatus = "revoked_identity"
	UnsupportedAlgorithm VerificationStatus = "unsupported_algorithm"
	UnsupportedFormat    VerificationStatus = "unsupported_format"
	MalformedPackage     VerificationStatus = "malformed_package"
	IncompleteProvenance VerificationStatus = "incomplete_provenance"
)

type Verification struct {
	FormatVersion  int                `json:"format_version"`
	Status         VerificationStatus `json:"status"`
	PackageID      string             `json:"package_id,omitempty"`
	SignatureValid bool               `json:"signature_valid"`
	Trust          TrustState         `json:"trust"`
	Identity       *Identity          `json:"identity,omitempty"`
	Manifest       *Manifest          `json:"manifest,omitempty"`
	Importable     bool               `json:"importable"`
	Warnings       []string           `json:"warnings"`
	Detail         string             `json:"detail"`
}

type ExportRequest struct {
	DatasetContentID string   `json:"dataset_content_id"`
	IdentityKeyID    string   `json:"identity_key_id"`
	IncludeReportIDs []string `json:"include_report_ids"`
}

type ExportResult struct {
	PackageID    string       `json:"package_id"`
	BytesBase64  string       `json:"bytes_base64"`
	Bytes        int          `json:"bytes"`
	Verification Verification `json:"verification"`
}

type ImportResult struct {
	Verification Verification             `json:"verification"`
	Dataset      governance.DatasetRecord `json:"dataset"`
	Reports      int                      `json:"reports"`
	Duplicate    bool                     `json:"duplicate"`
}
