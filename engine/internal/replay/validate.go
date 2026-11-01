package replay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"

	"chronosync.local/engine/internal/model"
)

func typed(code, detail string) error { return &Error{Code: code, Detail: detail} }

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}

func canonicalFixture(f Fixture) ([]byte, error) {
	raw, err := json.Marshal(f)
	if err != nil {
		return nil, typed("INVALID_FIXTURE", "fixture encoding")
	}
	return raw, nil
}

func replayID(f Fixture) (string, error) {
	f.ReplayID = ""
	raw, err := canonicalFixture(f)
	if err != nil {
		return "", err
	}
	return digest(raw)[:32], nil
}

func seal(f Fixture) ([]byte, error) {
	id, err := replayID(f)
	if err != nil {
		return nil, err
	}
	f.ReplayID = id
	raw, err := canonicalFixture(f)
	if err != nil {
		return nil, err
	}
	envelope, err := json.Marshal(Envelope{Fixture: f, SHA256: digest(raw)})
	if err != nil || len(envelope) > MaxFixtureBytes {
		return nil, typed("FIXTURE_TOO_LARGE", "canonical fixture exceeds limit")
	}
	return envelope, nil
}

func Load(raw []byte) (Fixture, error) {
	var envelope Envelope
	if len(raw) == 0 {
		return Fixture{}, typed("INVALID_FIXTURE", "empty fixture")
	}
	if len(raw) > MaxFixtureBytes {
		return Fixture{}, typed("FIXTURE_TOO_LARGE", "fixture exceeds limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return Fixture{}, typed("INVALID_FIXTURE", "malformed fixture")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return Fixture{}, typed("INVALID_FIXTURE", "trailing fixture data")
	}
	canonical, err := json.Marshal(envelope)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Fixture{}, typed("NON_CANONICAL_FIXTURE", "fixture bytes are not canonical")
	}
	fixtureRaw, err := canonicalFixture(envelope.Fixture)
	if err != nil || !validDigest(envelope.SHA256) || digest(fixtureRaw) != envelope.SHA256 {
		return Fixture{}, typed("INTEGRITY_MISMATCH", "fixture digest")
	}
	if err = validate(envelope.Fixture); err != nil {
		return Fixture{}, err
	}
	return envelope.Fixture, nil
}

func validate(f Fixture) error {
	if f.FormatVersion != FormatVersion {
		return typed("UNSUPPORTED_VERSION", "fixture format version")
	}
	if !model.ValidID(f.ReplayID) || !model.ValidID(f.SourceExecutionID) {
		return typed("INVALID_FIXTURE", "fixture identity")
	}
	wantID, err := replayID(f)
	if err != nil || wantID != f.ReplayID {
		return typed("INTEGRITY_MISMATCH", "replay identity")
	}
	last, err := strconv.ParseInt(f.SourceLastSequence, 10, 64)
	if err != nil || last < 1 || strconv.FormatInt(last, 10) != f.SourceLastSequence || f.SourceEventCount != int(last) {
		return typed("INVALID_FIXTURE", "source event range")
	}
	if f.SourceEventVersion != model.EventVersion || f.Adapter != "codex" || f.RedactionProfile != RedactionProfile {
		return typed("UNSUPPORTED_VERSION", "source or redaction contract")
	}
	if len(f.SupportedTypes) != 1 || f.SupportedTypes[0] != InteractionType {
		return typed("UNSUPPORTED_EVENT", "supported interaction declaration")
	}
	if len(f.Interactions) == 0 || len(f.Interactions) > MaxInteractions {
		return typed("INVALID_FIXTURE", "interaction count")
	}
	seen := map[string]bool{}
	var previous int64
	for _, interaction := range f.Interactions {
		sequence, parseErr := strconv.ParseInt(interaction.SourceSequence, 10, 64)
		if parseErr != nil || sequence <= previous || sequence > last || strconv.FormatInt(sequence, 10) != interaction.SourceSequence || interaction.Input.SourceSequence != interaction.SourceSequence {
			return typed("INVALID_FIXTURE", "interaction sequence")
		}
		previous = sequence
		if !model.ValidID(interaction.SourceEventID) || seen[interaction.SourceEventID] {
			return typed("INVALID_FIXTURE", "interaction identity")
		}
		seen[interaction.SourceEventID] = true
		if interaction.Type != InteractionType {
			return typed("UNSUPPORTED_EVENT", "interaction type")
		}
		if !validDigest(interaction.Input.CommandSHA256) || !validDigest(interaction.Input.ItemIDSHA256) || !validDigest(interaction.Expected.OutputSHA256) {
			return typed("INVALID_FIXTURE", "interaction digest")
		}
	}
	return nil
}
