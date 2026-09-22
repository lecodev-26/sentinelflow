package regions

import (
	"errors"
	"strings"
)

// Region representa una región geográfica
type Region string

const (
	US_East    Region = "us-east"
	US_West    Region = "us-west"
	EU_West    Region = "eu-west"
	EU_Central Region = "eu-central"
	AP_South   Region = "ap-south"
	AP_East    Region = "ap-east"
	Local      Region = "local"
)

// Policy define políticas por región
type Policy struct {
	Region              Region   `json:"region"`
	AllowedProviders    []string `json:"allowed_providers,omitempty"`
	BlockedProviders    []string `json:"blocked_providers,omitempty"`
	AllowedModels       []string `json:"allowed_models,omitempty"`
	RequireLocalStorage bool     `json:"require_local_storage"`
}

// Resolver resuelve políticas regionales
type Resolver struct {
	policies map[Region]*Policy
}

// NewResolver crea un nuevo resolver
func NewResolver() *Resolver {
	r := &Resolver{policies: make(map[Region]*Policy)}

	// Políticas por defecto
	r.policies[EU_West] = &Policy{
		Region:              EU_West,
		AllowedProviders:    []string{"openai", "anthropic", "local"},
		RequireLocalStorage: true, // GDPR
	}
	r.policies[EU_Central] = r.policies[EU_West]
	r.policies[US_East] = &Policy{
		Region:           US_East,
		AllowedProviders: []string{"openai", "anthropic", "google", "local"},
	}
	r.policies[US_West] = r.policies[US_East]

	return r
}

// SetPolicy configura una política
func (r *Resolver) SetPolicy(p *Policy) {
	r.policies[p.Region] = p
}

// GetPolicy devuelve la política de una región
func (r *Resolver) GetPolicy(region Region) (*Policy, bool) {
	p, exists := r.policies[region]
	return p, exists
}

// IsProviderAllowed verifica si un provider está permitido en una región
func (r *Resolver) IsProviderAllowed(region Region, provider string) bool {
	policy, exists := r.policies[region]
	if !exists {
		return true // sin política = permitir
	}

	// Si hay allowlist, el provider debe estar en ella
	if len(policy.AllowedProviders) > 0 {
		found := false
		for _, p := range policy.AllowedProviders {
			if p == provider {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Si está en la blocklist, denegar
	for _, p := range policy.BlockedProviders {
		if p == provider {
			return false
		}
	}

	return true
}

// IsModelAllowed verifica si un modelo está permitido en una región
func (r *Resolver) IsModelAllowed(region Region, model string) bool {
	policy, exists := r.policies[region]
	if !exists || len(policy.AllowedModels) == 0 {
		return true
	}

	for _, m := range policy.AllowedModels {
		if strings.HasPrefix(model, m) {
			return true
		}
	}
	return false
}

// DetectRegion detecta la región de una IP (simplificado)
func DetectRegion(ip string) Region {
	// En producción, usar una base de datos GeoIP
	if strings.HasPrefix(ip, "192.168") || strings.HasPrefix(ip, "10.") {
		return Local
	}
	if strings.HasPrefix(ip, "5.") || strings.HasPrefix(ip, "85.") {
		return EU_West
	}
	return US_East
}

// ResolveProvider dado un conjunto de candidatos, filtra por región
func (r *Resolver) ResolveProvider(region Region, candidates []string) (string, error) {
	for _, c := range candidates {
		if r.IsProviderAllowed(region, c) {
			return c, nil
		}
	}
	return "", errors.New("no allowed provider for region " + string(region))
}
