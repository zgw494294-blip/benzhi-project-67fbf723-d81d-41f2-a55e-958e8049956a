package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func EventDigest(event Event) string {
	event.Digest = ""
	return Digest(event)
}

func Digest(value any) string {
	encoded, _ := json.Marshal(value)
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:])
}
