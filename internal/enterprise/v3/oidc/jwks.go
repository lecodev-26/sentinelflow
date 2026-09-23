package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// JWK representa una clave pública en formato JSON Web Key
type JWK struct {
	KID string `json:"kid"`
	Kty string `json:"kty"` // "RSA"
	Use string `json:"use"` // "sig"
	Alg string `json:"alg"` // "RS256"
	N   string `json:"n"`   // modulus (base64url)
	E   string `json:"e"`   // exponent (base64url)
}

// JWKS es un conjunto de claves (JSON Web Key Set)
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWKSCache cachea las claves públicas con TTL
type JWKSCache struct {
	mu      sync.RWMutex
	entries map[Provider]*jwksEntry
}

type jwksEntry struct {
	keys      map[string]*rsa.PublicKey // kid -> public key
	fetchedAt time.Time
}

// NewJWKSCache crea un nuevo caché
func NewJWKSCache() *JWKSCache {
	return &JWKSCache{
		entries: make(map[Provider]*jwksEntry),
	}
}

// GetKey devuelve la clave pública para un provider + kid.
// Si no está en caché o expiró, la refetchea.
func (c *JWKSCache) GetKey(ctx context.Context, mgr *Manager, provider Provider, kid string) (*rsa.PublicKey, error) {
	// Intentar leer de caché
	c.mu.RLock()
	entry, exists := c.entries[provider]
	c.mu.RUnlock()

	if exists && time.Since(entry.fetchedAt) < 1*time.Hour {
		if key, ok := entry.keys[kid]; ok {
			return key, nil
		}
	}

	// Refetch
	keys, err := c.fetch(ctx, mgr, provider)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.entries[provider] = &jwksEntry{
		keys:      keys,
		fetchedAt: time.Now(),
	}
	c.mu.Unlock()

	if key, ok := keys[kid]; ok {
		return key, nil
	}
	return nil, fmt.Errorf("key %s not found in JWKS", kid)
}

// fetch obtiene las claves del endpoint JWKS del IdP
func (c *JWKSCache) fetch(ctx context.Context, mgr *Manager, provider Provider) (map[string]*rsa.PublicKey, error) {
	ep, ok := mgr.GetEndpoints(provider)
	if !ok || ep.JWKSURL == "" {
		return nil, errors.New("JWKS URL not configured for provider")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", ep.JWKSURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, fmt.Errorf("error parsing JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue
		}
		pubKey, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			continue
		}
		keys[jwk.KID] = pubKey
	}

	if len(keys) == 0 {
		return nil, errors.New("no valid RSA keys found in JWKS")
	}

	return keys, nil
}

// jwkToRSAPublicKey convierte un JWK en *rsa.PublicKey
func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decodificar modulus (base64url)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("error decoding modulus: %w", err)
	}

	// Decodificar exponent (base64url)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("error decoding exponent: %w", err)
	}

	// Convertir bytes a big.Int
	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}
