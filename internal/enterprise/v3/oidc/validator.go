package oidc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// IDTokenClaims son los claims estándar de un ID token OIDC.
// Implementa jwt.Claims (v5) mediante el embebido de jwt.RegisteredClaims.
type IDTokenClaims struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture,omitempty"`
	Nonce   string `json:"nonce,omitempty"`

	jwt.RegisteredClaims
}

// Validator valida ID tokens OIDC
type Validator struct {
	jwks *JWKSCache
}

// NewValidator crea un nuevo validador
func NewValidator(jwks *JWKSCache) *Validator {
	return &Validator{jwks: jwks}
}

// Validate valida un ID token:
//  1. Parsea el JWT
//  2. Verifica la firma con la clave pública del JWKS
//  3. Verifica el issuer
//  4. Verifica el audience
//  5. Verifica la expiración
//  6. Verifica el nonce (si aplica)
func (v *Validator) Validate(
	ctx context.Context,
	mgr *Manager,
	provider Provider,
	idToken string,
	expectedNonce string,
) (*IDTokenClaims, error) {
	cfg, ok := mgr.GetConfig(provider)
	if !ok {
		return nil, errors.New("provider not registered")
	}

	claims := &IDTokenClaims{}

	// Parsear con validación de firma
	token, err := jwt.ParseWithClaims(idToken, claims, func(token *jwt.Token) (interface{}, error) {
		// Verificar algoritmo
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Obtener kid del header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid not found in token header")
		}

		// Obtener clave pública del JWKS
		return v.jwks.GetKey(ctx, mgr, provider, kid)
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token is not valid")
	}

	// Verificar issuer (si está configurado)
	if cfg.Issuer != "" && claims.Issuer != cfg.Issuer {
		return nil, fmt.Errorf("invalid issuer: got %s, expected %s", claims.Issuer, cfg.Issuer)
	}

	// Verificar audience
	if len(claims.Audience) == 0 || claims.Audience[0] != cfg.ClientID {
		return nil, fmt.Errorf("invalid audience: got %v, expected %s", claims.Audience, cfg.ClientID)
	}

	// Verificar expiración (jwt lib ya lo hace, pero doble check)
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, errors.New("token expired")
	}

	// Verificar nonce si lo esperábamos
	if expectedNonce != "" && claims.Nonce != expectedNonce {
		return nil, fmt.Errorf("invalid nonce: got %s, expected %s", claims.Nonce, expectedNonce)
	}

	return claims, nil
}
