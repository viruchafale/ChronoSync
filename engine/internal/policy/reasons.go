package policy

// Message provides fixed actionable text without echoing private request data.
func (r Reason) Message() string {
	switch r.Code {
	case InvalidPolicy:
		return "Supply a versioned policy with valid scope identities and explicit, well-formed restrictions."
	case UnsupportedRequirement:
		return "Remove no restrictions implicitly; obtain a reviewed contract for the unsupported requirement."
	case MissingRequirement:
		return "Specify this dimension explicitly; omission cannot grant permission."
	case ConflictingRequirement:
		return "Specify exactly one consistent restriction for this dimension."
	case InvalidTarget:
		return "Supply a supported target identity and an explicit evaluation time."
	case InvalidCapability:
		return "Obtain well-formed, unambiguous evidence from a trusted capability provider."
	case MissingCapability:
		return "Obtain capability evidence for the required restriction before admission."
	case CapabilityUnavailable:
		return "The required capability is unavailable; do not launch under this policy."
	case CapabilityUnknown:
		return "The required capability has not been established; verified evidence is required."
	case CapabilityUnsupported:
		return "This environment does not support the required capability; select a reviewed enforcing backend."
	case UnverifiedEvidence:
		return "Configuration or intent alone is insufficient; obtain verified enforcement evidence."
	case StaleEvidence:
		return "Revalidate capability evidence for the current evaluation time."
	case WrongTarget:
		return "Obtain evidence for this exact platform, backend version and configuration."
	case WrongScope:
		return "Obtain evidence bound to this workspace and executable identity."
	case InsufficientEnforcement:
		return "Observation, advice or partial coverage cannot satisfy required enforcement."
	case RestrictionMismatch:
		return "Obtain evidence for the exact restriction; never silently widen it."
	default:
		return "Policy refused; review the unsupported decision code."
	}
}
