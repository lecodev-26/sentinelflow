package idgen

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// RandomHex devuelve n bytes aleatorios en hex (2n chars)
func RandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// RandomBase64URL devuelve n bytes aleatorios en base64 URL-safe
func RandomBase64URL(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// NewID devuelve un ID con prefijo + 16 bytes crypto rand
// Ejemplo: NewID("user") → "user_3f8a2b1c..."
func NewID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, RandomHex(8))
}

// NewUUIDv4 devuelve un UUID v4 estándar
func NewUUIDv4() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}

	// Version 4: byte 6 = 0100xxxx
	b[6] = (b[6] & 0x0f) | 0x40
	// Variant RFC 4122: byte 8 = 10xxxxxx
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
