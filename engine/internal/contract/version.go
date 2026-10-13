// Package contract defines the implemented foundation compatibility report.
package contract

const ProtocolVersion = 1

type Version struct {
	ProtocolVersion int      `json:"protocol_version"`
	EngineVersion   string   `json:"engine_version"`
	Capabilities    []string `json:"capabilities"`
}

func CurrentVersion() Version {
	return Version{ProtocolVersion: ProtocolVersion, EngineVersion: "0.3.0", Capabilities: []string{"version", "stdio-recorder-v1", "event-v2", "timeline-v1", "snapshots-v1"}}
}
