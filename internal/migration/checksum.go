package migration

import (
	"crypto/sha256"
	"encoding/hex"
)

// Checksum returns the SHA-256 hex digest for migration file content.
func Checksum(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}
