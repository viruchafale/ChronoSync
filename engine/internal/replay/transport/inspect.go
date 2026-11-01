package transport

type Inspection struct {
	FormatVersion int               `json:"format_version"`
	Domain        string            `json:"domain"`
	Interactions  int               `json:"interactions"`
	Responses     int               `json:"responses"`
	Supported     []string          `json:"supported"`
	Entries       []InspectionEntry `json:"entries"`
}

type InspectionEntry struct {
	Sequence            uint64            `json:"sequence"`
	Occurrence          uint32            `json:"occurrence"`
	Transport           TransportIdentity `json:"transport"`
	RequestFingerprint  string            `json:"request_fingerprint"`
	ResponseReference   string            `json:"response_reference"`
	ResponseFingerprint string            `json:"response_fingerprint"`
	ResponseStatus      int               `json:"response_status"`
}

func Inspect(raw []byte) (Inspection, error) {
	recording, err := LoadTransportRecording(raw)
	if err != nil {
		return Inspection{}, err
	}
	inspection := Inspection{
		FormatVersion: TransportFixtureFormatVersion,
		Domain:        TransportFixtureDomain,
		Interactions:  len(recording.entries),
		Responses:     len(recording.responses),
		Supported:     []string{"synthetic.request.v1", "synthetic.response.v1"},
		Entries:       make([]InspectionEntry, 0, len(recording.entries)),
	}
	for _, entry := range recording.entries {
		response, ok := (&TransportMatcher{recording: recording}).response(entry.ResponseReference)
		if !ok {
			return Inspection{}, typed("INVALID_TRANSPORT_FIXTURE", "missing response reference")
		}
		inspection.Entries = append(inspection.Entries, InspectionEntry{
			Sequence: entry.Sequence, Occurrence: entry.Occurrence, Transport: entry.Transport,
			RequestFingerprint: entry.RequestFingerprint, ResponseReference: entry.ResponseReference,
			ResponseFingerprint: response.Fingerprint, ResponseStatus: response.Response.Status,
		})
	}
	return inspection, nil
}
