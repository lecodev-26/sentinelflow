package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TokenResponse es la respuesta del endpoint /token del IdP
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope,omitempty"`
}

// ExchangeCode intercambia el authorization code por tokens.
// POST al endpoint /token del IdP con:
//   - grant_type=authorization_code
//   - code=<code>
//   - redirect_uri=<redirect>
//   - client_id=<client_id>
//   - client_secret=<client_secret>
func (f *Flow) ExchangeCode(ctx context.Context, provider Provider, code string) (*TokenResponse, error) {
	cfg, ok := f.manager.GetConfig(provider)
	if !ok {
		return nil, errors.New("provider not registered")
	}
	ep, ok := f.manager.GetEndpoints(provider)
	if !ok {
		return nil, errors.New("provider endpoints not found")
	}

	// Construir body
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURL)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", ep.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error exchanging code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokens TokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("error parsing token response: %w", err)
	}

	if tokens.IDToken == "" {
		return nil, errors.New("no id_token in response")
	}

	return &tokens, nil
}
