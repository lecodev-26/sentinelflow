package events

import (
	"crypto/rand"
	"encoding/hex"
)

func generateEventID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return "evt_" + hex.EncodeToString(b)
}
