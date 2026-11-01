package transport

import (
	"bytes"
	"encoding/json"
	"io"
)

const (
	TransportFixtureDomain   = "chronosync.synthetic-transport"
	MaxTransportFixtureBytes = 1024 * 1024
)

type transportFixtureEnvelope struct {
	Fixture transportFixtureWire `json:"fixture"`
	SHA256  string               `json:"sha256"`
}

type transportFixtureWire struct {
	FormatVersion              int                     `json:"format_version"`
	Domain                     string                  `json:"domain"`
	RequestFingerprintVersion  int                     `json:"request_fingerprint_version"`
	ResponseFingerprintVersion int                     `json:"response_fingerprint_version"`
	Entries                    []transportEntryWire    `json:"entries"`
	Responses                  []transportResponseWire `json:"responses"`
}

type transportEntryWire struct {
	Sequence           uint64            `json:"sequence"`
	Occurrence         uint32            `json:"occurrence"`
	Transport          TransportIdentity `json:"transport"`
	RequestFingerprint string            `json:"request_fingerprint"`
	ResponseReference  string            `json:"response_reference"`
}

type transportResponseWire struct {
	Reference   string              `json:"reference"`
	Fingerprint string              `json:"fingerprint"`
	Status      int                 `json:"status"`
	Metadata    []TransportMetadata `json:"metadata"`
}

func (recording TransportRecording) Bytes() ([]byte, error) {
	if len(recording.entries) == 0 || len(recording.entries) > MaxTransportInteractions || len(recording.responses) == 0 || len(recording.responses) > len(recording.entries) {
		return nil, typed("INVALID_TRANSPORT_FIXTURE", "recording count")
	}
	wire := transportFixtureWire{
		FormatVersion: TransportFixtureFormatVersion, Domain: TransportFixtureDomain,
		RequestFingerprintVersion: RequestFingerprintVersion, ResponseFingerprintVersion: ResponseFingerprintVersion,
		Entries: make([]transportEntryWire, len(recording.entries)), Responses: make([]transportResponseWire, len(recording.responses)),
	}
	for index, entry := range recording.entries {
		wire.Entries[index] = transportEntryWire{entry.Sequence, entry.Occurrence, entry.Transport, entry.RequestFingerprint, entry.ResponseReference}
	}
	for index, response := range recording.responses {
		wire.Responses[index] = transportResponseWire{response.Reference, response.Fingerprint, response.Response.Status, append([]TransportMetadata(nil), response.Response.Metadata...)}
	}
	fixtureBytes, err := json.Marshal(wire)
	if err != nil {
		return nil, typed("INVALID_TRANSPORT_FIXTURE", "fixture encoding")
	}
	raw, err := json.Marshal(transportFixtureEnvelope{Fixture: wire, SHA256: digest(fixtureBytes)})
	if err != nil || len(raw) > MaxTransportFixtureBytes {
		return nil, typed("TRANSPORT_FIXTURE_TOO_LARGE", "fixture exceeds limit")
	}
	return raw, nil
}

func LoadTransportRecording(raw []byte) (TransportRecording, error) {
	if len(raw) == 0 {
		return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "empty fixture")
	}
	if len(raw) > MaxTransportFixtureBytes {
		return TransportRecording{}, typed("TRANSPORT_FIXTURE_TOO_LARGE", "fixture exceeds limit")
	}
	var envelope transportFixtureEnvelope
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "malformed fixture")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "trailing fixture data")
	}
	canonicalEnvelope, err := json.Marshal(envelope)
	if err != nil || !canonicalEqual(canonicalEnvelope, raw) {
		return TransportRecording{}, typed("NON_CANONICAL_TRANSPORT_FIXTURE", "fixture bytes are not canonical")
	}
	fixtureBytes, err := json.Marshal(envelope.Fixture)
	if err != nil || !validDigest(envelope.SHA256) || digest(fixtureBytes) != envelope.SHA256 {
		return TransportRecording{}, typed("TRANSPORT_INTEGRITY_MISMATCH", "fixture digest")
	}
	return recordingFromWire(envelope.Fixture)
}

func recordingFromWire(wire transportFixtureWire) (TransportRecording, error) {
	if wire.FormatVersion != TransportFixtureFormatVersion || wire.Domain != TransportFixtureDomain || wire.RequestFingerprintVersion != RequestFingerprintVersion || wire.ResponseFingerprintVersion != ResponseFingerprintVersion {
		return TransportRecording{}, typed("UNSUPPORTED_TRANSPORT_VERSION", "fixture contract")
	}
	if len(wire.Entries) == 0 || len(wire.Entries) > MaxTransportInteractions || len(wire.Responses) == 0 || len(wire.Responses) > len(wire.Entries) {
		return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "fixture count")
	}
	responseByReference := make(map[string]recordedResponse, len(wire.Responses))
	previousReference := ""
	for _, item := range wire.Responses {
		if item.Reference <= previousReference || item.Reference == "" || !validDigest(item.Fingerprint) {
			return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "response identity or order")
		}
		previousReference = item.Reference
		normalized, fingerprint, err := fingerprintResponse(SyntheticResponse{Status: item.Status, Metadata: item.Metadata})
		if err != nil || fingerprint != item.Fingerprint || item.Reference != "rsp_"+fingerprint[:32] {
			return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "response fingerprint")
		}
		responseByReference[item.Reference] = recordedResponse{Reference: item.Reference, Fingerprint: fingerprint, Response: normalized}
	}
	recording := TransportRecording{entries: make([]transportEntry, 0, len(wire.Entries)), responses: make([]recordedResponse, 0, len(wire.Responses))}
	occurrences := make(map[string]uint32)
	referencedResponses := make(map[string]bool, len(wire.Responses))
	for index, item := range wire.Entries {
		if item.Sequence != uint64(index+1) || !validDigest(item.RequestFingerprint) {
			return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "interaction sequence or fingerprint")
		}
		identity, err := normalizeIdentity(item.Transport)
		if err != nil || identity != item.Transport {
			return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "transport identity")
		}
		key := matchKey(identity, item.RequestFingerprint)
		if item.Occurrence != occurrences[key]+1 {
			return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "interaction occurrence")
		}
		occurrences[key] = item.Occurrence
		if _, ok := responseByReference[item.ResponseReference]; !ok {
			return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "missing response reference")
		}
		referencedResponses[item.ResponseReference] = true
		recording.entries = append(recording.entries, transportEntry{item.Sequence, item.Occurrence, identity, item.RequestFingerprint, item.ResponseReference})
	}
	for _, item := range wire.Responses {
		if !referencedResponses[item.Reference] {
			return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "unreferenced response")
		}
		recording.responses = append(recording.responses, responseByReference[item.Reference])
	}
	return cloneRecording(recording), nil
}
