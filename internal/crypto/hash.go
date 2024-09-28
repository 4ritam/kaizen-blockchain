package crypto

import (
	"encoding/hex"
	"regexp"
)

// ValidateHash checks if a given hash is a valid SHA-512 hash
func ValidateHash(hash string) bool {
    if len(hash) != 128 {
        return false
    }

    match, _ := regexp.MatchString("^[a-fA-F0-9]+$", hash)
    return match
}

// HashToBytes converts a hexadecimal hash string to a byte slice
func HashToBytes(hash string) ([]byte, error) {
    return hex.DecodeString(hash)
}

// BytesToHash converts a byte slice to a hexadecimal hash string
func BytesToHash(b []byte) string {
    return hex.EncodeToString(b)
}
