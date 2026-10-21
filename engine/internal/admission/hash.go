package admission

import (
	"crypto/sha256"
	"encoding/hex"
)

func referenceHash(domain string, d Digest) Digest {
	h := sha256.Sum256(append([]byte(domain), []byte(d)...))
	return Digest(hex.EncodeToString(h[:]))
}
