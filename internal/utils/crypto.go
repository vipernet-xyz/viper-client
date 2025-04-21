package utils

import (
	"encoding/hex"

	"golang.org/x/crypto/sha3"
)

// SHA3Hash generates a SHA3-256 hash for the given input as a hex string
func SHA3Hash(input string) string {
	h := sha3.New256()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// KeyExists checks if a key exists in a map
func KeyExists(m map[string]interface{}, key string) bool {
	_, ok := m[key]
	return ok
}
