package transport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"sync"

	"chronosync.local/engine/internal/redact"
)

type Error struct {
	Code   string
	Detail string
}

func (err *Error) Error() string { return err.Code + ": " + err.Detail }

func typed(code, detail string) error { return &Error{Code: code, Detail: detail} }

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}

const (
	TransportFixtureFormatVersion = 1
	RequestFingerprintVersion     = 1
	ResponseFingerprintVersion    = 1
	MaxTransportInteractions      = 10_000
	MaxTransportMetadataFields    = 16
	MaxTransportMetadataValue     = 256
)

var transportToken = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/+;=-]{0,255}$`)

var requestMetadataNames = map[string]bool{
	"accept": true, "content-type": true, "operation-kind": true, "tool-name": true,
}

var responseMetadataNames = map[string]bool{
	"content-type": true, "finish-reason": true, "result-kind": true,
}

type TransportIdentity struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Version  string `json:"version"`
}

type TransportMetadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SyntheticRequest is inert. Body is present only so unsupported body-bearing
// callers can be refused explicitly; it is never fingerprinted or persisted.
type SyntheticRequest struct {
	Transport  TransportIdentity   `json:"transport"`
	Operation  string              `json:"operation"`
	Metadata   []TransportMetadata `json:"metadata"`
	Occurrence uint32              `json:"occurrence"`
	Body       []byte              `json:"-"`
}

// SyntheticResponse contains only bounded, allowlisted metadata. Arbitrary
// response bodies are outside the Phase 5B contract.
type SyntheticResponse struct {
	Status   int                 `json:"status"`
	Metadata []TransportMetadata `json:"metadata"`
	Body     []byte              `json:"-"`
}

type TransportInteractionDraft struct {
	Sequence   uint64
	Occurrence uint32
	Request    SyntheticRequest
	Response   SyntheticResponse
}

type TransportOutcomeKind string

const (
	TransportMatched     TransportOutcomeKind = "matched"
	TransportUnmatched   TransportOutcomeKind = "unmatched"
	TransportAmbiguous   TransportOutcomeKind = "ambiguous"
	TransportUnsupported TransportOutcomeKind = "unsupported"
	TransportCancelled   TransportOutcomeKind = "cancelled"
	TransportInvalid     TransportOutcomeKind = "invalid"
)

type TransportOutcome struct {
	Kind                TransportOutcomeKind `json:"kind"`
	ResponseReference   string               `json:"response_reference,omitempty"`
	ResponseFingerprint string               `json:"response_fingerprint,omitempty"`
	Response            SyntheticResponse    `json:"response"`
	Code                string               `json:"code,omitempty"`
}

type SyntheticTransport interface {
	Exchange(context.Context, SyntheticRequest) TransportOutcome
}

type transportEntry struct {
	Sequence           uint64
	Occurrence         uint32
	Transport          TransportIdentity
	RequestFingerprint string
	ResponseReference  string
}

type recordedResponse struct {
	Reference   string
	Fingerprint string
	Response    SyntheticResponse
}

// TransportRecording owns all nested values. It exposes recordings only as
// detached canonical bytes or through a matcher that takes another copy.
type TransportRecording struct {
	entries   []transportEntry
	responses []recordedResponse
}

type requestFingerprintBody struct {
	FormatVersion int                 `json:"format_version"`
	Transport     TransportIdentity   `json:"transport"`
	Operation     string              `json:"operation"`
	Metadata      []TransportMetadata `json:"metadata"`
}

type responseFingerprintBody struct {
	FormatVersion int                 `json:"format_version"`
	Status        int                 `json:"status"`
	Metadata      []TransportMetadata `json:"metadata"`
}

func NewTransportRecording(drafts []TransportInteractionDraft) (TransportRecording, error) {
	if len(drafts) == 0 || len(drafts) > MaxTransportInteractions {
		return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "interaction count")
	}
	recording := TransportRecording{entries: make([]transportEntry, 0, len(drafts))}
	occurrences := make(map[string]uint32)
	responses := make(map[string]recordedResponse)
	for index, draft := range drafts {
		if draft.Sequence != uint64(index+1) {
			return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "interaction sequence")
		}
		identity, requestFingerprint, err := fingerprintRequest(draft.Request)
		if err != nil {
			return TransportRecording{}, err
		}
		response, responseFingerprint, err := fingerprintResponse(draft.Response)
		if err != nil {
			return TransportRecording{}, err
		}
		key := matchKey(identity, requestFingerprint)
		wantOccurrence := occurrences[key] + 1
		if draft.Occurrence != wantOccurrence {
			return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "interaction occurrence")
		}
		occurrences[key] = wantOccurrence
		reference := "rsp_" + responseFingerprint[:32]
		if existing, ok := responses[reference]; ok && existing.Fingerprint != responseFingerprint {
			return TransportRecording{}, typed("INVALID_TRANSPORT_FIXTURE", "response reference collision")
		}
		if _, ok := responses[reference]; !ok {
			responses[reference] = recordedResponse{Reference: reference, Fingerprint: responseFingerprint, Response: cloneResponse(response)}
		}
		recording.entries = append(recording.entries, transportEntry{
			Sequence: draft.Sequence, Occurrence: draft.Occurrence, Transport: identity,
			RequestFingerprint: requestFingerprint, ResponseReference: reference,
		})
	}
	refs := make([]string, 0, len(responses))
	for reference := range responses {
		refs = append(refs, reference)
	}
	sort.Strings(refs)
	for _, reference := range refs {
		recording.responses = append(recording.responses, responses[reference])
	}
	return cloneRecording(recording), nil
}

func fingerprintRequest(request SyntheticRequest) (TransportIdentity, string, error) {
	if len(request.Body) != 0 {
		return TransportIdentity{}, "", typed("UNSUPPORTED_INTERACTION", "request body")
	}
	identity, err := normalizeIdentity(request.Transport)
	if err != nil {
		return TransportIdentity{}, "", err
	}
	operation, err := safeTransportValue("operation", request.Operation)
	if err != nil {
		return TransportIdentity{}, "", err
	}
	metadata, err := normalizeMetadata(request.Metadata, requestMetadataNames)
	if err != nil {
		return TransportIdentity{}, "", err
	}
	raw, err := json.Marshal(requestFingerprintBody{RequestFingerprintVersion, identity, operation, metadata})
	if err != nil {
		return TransportIdentity{}, "", typed("INVALID_INTERACTION", "request encoding")
	}
	return identity, digest(raw), nil
}

func fingerprintResponse(response SyntheticResponse) (SyntheticResponse, string, error) {
	if len(response.Body) != 0 {
		return SyntheticResponse{}, "", typed("UNSUPPORTED_INTERACTION", "response body")
	}
	if response.Status < 100 || response.Status > 599 {
		return SyntheticResponse{}, "", typed("INVALID_INTERACTION", "response status")
	}
	metadata, err := normalizeMetadata(response.Metadata, responseMetadataNames)
	if err != nil {
		return SyntheticResponse{}, "", err
	}
	normalized := SyntheticResponse{Status: response.Status, Metadata: metadata}
	raw, err := json.Marshal(responseFingerprintBody{ResponseFingerprintVersion, normalized.Status, normalized.Metadata})
	if err != nil {
		return SyntheticResponse{}, "", typed("INVALID_INTERACTION", "response encoding")
	}
	return normalized, digest(raw), nil
}

func normalizeIdentity(identity TransportIdentity) (TransportIdentity, error) {
	name, err := safeTransportValue("transport name", identity.Name)
	if err != nil {
		return TransportIdentity{}, err
	}
	protocol, err := safeTransportValue("transport protocol", identity.Protocol)
	if err != nil {
		return TransportIdentity{}, err
	}
	version, err := safeTransportValue("transport version", identity.Version)
	if err != nil {
		return TransportIdentity{}, err
	}
	return TransportIdentity{Name: name, Protocol: protocol, Version: version}, nil
}

func normalizeMetadata(input []TransportMetadata, allowed map[string]bool) ([]TransportMetadata, error) {
	if len(input) > MaxTransportMetadataFields {
		return nil, typed("INVALID_INTERACTION", "metadata count")
	}
	output := make([]TransportMetadata, len(input))
	seen := make(map[string]bool, len(input))
	for index, field := range input {
		name := strings.ToLower(field.Name)
		if !allowed[name] || seen[name] || field.Name != name {
			return nil, typed("UNSUPPORTED_INTERACTION", "metadata name")
		}
		value, err := safeTransportValue("metadata value", field.Value)
		if err != nil {
			return nil, err
		}
		seen[name] = true
		output[index] = TransportMetadata{Name: name, Value: value}
	}
	sort.Slice(output, func(i, j int) bool { return output[i].Name < output[j].Name })
	return output, nil
}

func safeTransportValue(field, value string) (string, error) {
	if value == "" || len(value) > MaxTransportMetadataValue || !transportToken.MatchString(value) {
		return "", typed("UNSUPPORTED_INTERACTION", field)
	}
	redacted := redact.Text(value)
	if redacted != value || strings.Contains(redacted, "[REDACTED]") || strings.Contains(redacted, "[TRUNCATED]") {
		return "", typed("UNSUPPORTED_INTERACTION", field+" redaction")
	}
	return redacted, nil
}

func matchKey(identity TransportIdentity, requestFingerprint string) string {
	return identity.Name + "\x00" + identity.Protocol + "\x00" + identity.Version + "\x00" + requestFingerprint
}

type TransportMatcher struct {
	mu        sync.Mutex
	recording TransportRecording
	consumed  map[string]uint32
}

func NewTransportMatcher(recording TransportRecording) *TransportMatcher {
	return &TransportMatcher{recording: cloneRecording(recording), consumed: make(map[string]uint32)}
}

func (matcher *TransportMatcher) Exchange(ctx context.Context, request SyntheticRequest) TransportOutcome {
	if err := ctx.Err(); err != nil {
		return TransportOutcome{Kind: TransportCancelled, Code: "CANCELLED"}
	}
	identity, requestFingerprint, err := fingerprintRequest(request)
	if err != nil {
		return transportErrorOutcome(err)
	}
	key := matchKey(identity, requestFingerprint)
	matcher.mu.Lock()
	defer matcher.mu.Unlock()
	if err = ctx.Err(); err != nil {
		return TransportOutcome{Kind: TransportCancelled, Code: "CANCELLED"}
	}
	candidates := make([]transportEntry, 0, 1)
	for _, entry := range matcher.recording.entries {
		if entry.Transport == identity && entry.RequestFingerprint == requestFingerprint {
			candidates = append(candidates, entry)
		}
	}
	if len(candidates) == 0 {
		return TransportOutcome{Kind: TransportUnmatched, Code: "NO_RECORDED_REQUEST"}
	}
	occurrence := request.Occurrence
	if occurrence == 0 {
		if len(candidates) != 1 {
			return TransportOutcome{Kind: TransportAmbiguous, Code: "OCCURRENCE_REQUIRED"}
		}
		occurrence = candidates[0].Occurrence
	}
	want := matcher.consumed[key] + 1
	if occurrence != want {
		return TransportOutcome{Kind: TransportInvalid, Code: "INVALID_OCCURRENCE_ORDER"}
	}
	eligible := make([]transportEntry, 0, 1)
	for _, entry := range candidates {
		if entry.Occurrence == occurrence {
			eligible = append(eligible, entry)
		}
	}
	if len(eligible) == 0 {
		return TransportOutcome{Kind: TransportUnmatched, Code: "NO_RECORDED_OCCURRENCE"}
	}
	if len(eligible) != 1 {
		return TransportOutcome{Kind: TransportAmbiguous, Code: "AMBIGUOUS_RECORDED_OCCURRENCE"}
	}
	entry := eligible[0]
	response, ok := matcher.response(entry.ResponseReference)
	if !ok {
		return TransportOutcome{Kind: TransportInvalid, Code: "MISSING_RESPONSE_REFERENCE"}
	}
	if err = ctx.Err(); err != nil {
		return TransportOutcome{Kind: TransportCancelled, Code: "CANCELLED"}
	}
	matcher.consumed[key] = occurrence
	return TransportOutcome{
		Kind: TransportMatched, ResponseReference: response.Reference,
		ResponseFingerprint: response.Fingerprint, Response: cloneResponse(response.Response),
	}
}

func (matcher *TransportMatcher) Reset() {
	matcher.mu.Lock()
	defer matcher.mu.Unlock()
	matcher.consumed = make(map[string]uint32)
}

func (matcher *TransportMatcher) response(reference string) (recordedResponse, bool) {
	for _, response := range matcher.recording.responses {
		if response.Reference == reference {
			return response, true
		}
	}
	return recordedResponse{}, false
}

func transportErrorOutcome(err error) TransportOutcome {
	typedError, ok := err.(*Error)
	if !ok {
		return TransportOutcome{Kind: TransportInvalid, Code: "INVALID_INTERACTION"}
	}
	if typedError.Code == "UNSUPPORTED_INTERACTION" {
		return TransportOutcome{Kind: TransportUnsupported, Code: typedError.Code}
	}
	return TransportOutcome{Kind: TransportInvalid, Code: typedError.Code}
}

func cloneResponse(response SyntheticResponse) SyntheticResponse {
	return SyntheticResponse{Status: response.Status, Metadata: append([]TransportMetadata(nil), response.Metadata...)}
}

func cloneRecording(recording TransportRecording) TransportRecording {
	clone := TransportRecording{entries: append([]transportEntry(nil), recording.entries...), responses: make([]recordedResponse, len(recording.responses))}
	for index, response := range recording.responses {
		clone.responses[index] = recordedResponse{Reference: response.Reference, Fingerprint: response.Fingerprint, Response: cloneResponse(response.Response)}
	}
	return clone
}

func canonicalEqual(left, right []byte) bool { return bytes.Equal(left, right) }
