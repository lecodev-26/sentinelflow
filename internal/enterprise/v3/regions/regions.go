package regions

import (
	"errors"
	"strings"
)

// Region representa una región geográfica
type Region string

const (
	USEast    Region = "us-east"
	USWest    Region = "us-west"
	EUWest    Region = "eu-west"
	EUCentral Region = "eu-central"
	APSouth   Region = "ap-south"
	APEast    Region = "ap-east"
	Local     Region = "local"
)

// ResidencyLevel es el nivel de residencia de datos
type ResidencyLevel string

const (
	ResidencyGlobal ResidencyLevel = "global" // sin restricciones
	ResidencyEU     ResidencyLevel = "eu"     // solo EU
	ResidencyUS     ResidencyLevel = "us"     // solo US
	ResidencyAPAC   ResidencyLevel = "apac"   // solo APAC
)

// Policy define reglas por región/residencia
type Policy struct {
	Region              Region         `json:"region,omitempty"`
	Residency           ResidencyLevel `json:"residency"`
	AllowedProviders    []string       `json:"allowed_providers,omitempty"`
	BlockedProviders    []string       `json:"blocked_providers,omitempty"`
	AllowedRegions      []Region       `json:"allowed_regions,omitempty"`
	RequireLocalStorage bool           `json:"require_local_storage"`
	RetentionDays       int            `json:"retention_days,omitempty"`
}

// Resolver resuelve políticas de residencia + región
type Resolver struct {
	policies map[ResidencyLevel]*Policy
}

// NewResolver crea un resolver con políticas por defecto
func NewResolver() *Resolver {
	r := &Resolver{policies: make(map[ResidencyLevel]*Policy)}

	// GDPR - EU residency
	r.policies[ResidencyEU] = &Policy{
		Residency:           ResidencyEU,
		AllowedRegions:      []Region{EUWest, EUCentral},
		AllowedProviders:    []string{"openai", "anthropic", "mistral", "ollama"},
		RequireLocalStorage: true,
		RetentionDays:       30,
	}

	// US residency
	r.policies[ResidencyUS] = &Policy{
		Residency:        ResidencyUS,
		AllowedRegions:   []Region{USEast, USWest},
		AllowedProviders: []string{"openai", "anthropic", "google"},
		RetentionDays:    90,
	}

	// APAC residency
	r.policies[ResidencyAPAC] = &Policy{
		Residency:      ResidencyAPAC,
		AllowedRegions: []Region{APSouth, APEast},
		RetentionDays:  60,
	}

	// Global (sin restricciones)
	r.policies[ResidencyGlobal] = &Policy{
		Residency:     ResidencyGlobal,
		RetentionDays: 90,
	}

	return r
}

// SetPolicy configura una política
func (r *Resolver) SetPolicy(p *Policy) {
	r.policies[p.Residency] = p
}

// GetPolicy devuelve la política de un nivel de residencia
func (r *Resolver) GetPolicy(level ResidencyLevel) (*Policy, bool) {
	p, ok := r.policies[level]
	return p, ok
}

// ValidateProvider verifica si un provider está permitido para una residencia
func (r *Resolver) ValidateProvider(residency ResidencyLevel, provider string) error {
	policy, ok := r.policies[residency]
	if !ok {
		return nil // sin política, permitir
	}

	// Si hay allowlist, debe estar en ella
	if len(policy.AllowedProviders) > 0 {
		found := false
		for _, p := range policy.AllowedProviders {
			if p == provider {
				found = true
				break
			}
		}
		if !found {
			return errors.New("provider not allowed for residency " + string(residency) + ": " + provider)
		}
	}

	// Si está en blocklist, denegar
	for _, p := range policy.BlockedProviders {
		if p == provider {
			return errors.New("provider blocked for residency " + string(residency) + ": " + provider)
		}
	}

	return nil
}

// ValidateRegion verifica si una región está permitida
func (r *Resolver) ValidateRegion(residency ResidencyLevel, region Region) error {
	policy, ok := r.policies[residency]
	if !ok || len(policy.AllowedRegions) == 0 {
		return nil // sin restricción
	}

	for _, reg := range policy.AllowedRegions {
		if reg == region {
			return nil
		}
	}
	return errors.New("region not allowed for residency " + string(residency) + ": " + string(region))
}

// DetectRegion detecta una región a partir de una IP (simplificado)
func DetectRegion(ip string) Region {
	if strings.HasPrefix(ip, "127.") || strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") {
		return Local
	}
	// Simplificado: en producción usar GeoIP
	if strings.HasPrefix(ip, "5.") || strings.HasPrefix(ip, "85.") || strings.HasPrefix(ip, "91.") {
		return EUWest
	}
	if strings.HasPrefix(ip, "13.") || strings.HasPrefix(ip, "52.") {
		return USEast
	}
	if strings.HasPrefix(ip, "103.") || strings.HasPrefix(ip, "104.") {
		return APSouth
	}
	return USEast // default
}

// FilterProviders filtra una lista de providers según la residencia
func (r *Resolver) FilterProviders(residency ResidencyLevel, providers []string) []string {
	policy, ok := r.policies[residency]
	if !ok || len(policy.AllowedProviders) == 0 {
		return providers // sin restricción
	}

	allowed := make(map[string]bool)
	for _, p := range policy.AllowedProviders {
		allowed[p] = true
	}

	var result []string
	for _, p := range providers {
		if allowed[p] {
			result = append(result, p)
		}
	}
	return result
}

// List devuelve todas las políticas
func (r *Resolver) List() []*Policy {
	result := make([]*Policy, 0, len(r.policies))
	for _, p := range r.policies {
		result = append(result, p)
	}
	return result
}
