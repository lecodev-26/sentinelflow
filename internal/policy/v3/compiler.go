package v3

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// CompiledPolicy es una política compilada lista para evaluar rápido
type CompiledPolicy struct {
	Source        *Policy
	DenyPatterns  []CompiledRule
	AllowPatterns []CompiledRule
}

// CompiledRule es una regla compilada con regex precalculada
type CompiledRule struct {
	Original      Rule
	ModelRegex    *regexp.Regexp
	ProviderRegex *regexp.Regexp
	HasModel      bool
	HasProvider   bool
}

// Compiler compila políticas
type Compiler struct{}

// NewCompiler crea un compilador
func NewCompiler() *Compiler {
	return &Compiler{}
}

// Compile compila una política
func (c *Compiler) Compile(p *Policy) (*CompiledPolicy, error) {
	if p == nil {
		return nil, errors.New("nil policy")
	}
	if p.ID == "" {
		return nil, errors.New("policy ID is required")
	}

	compiled := &CompiledPolicy{
		Source: p,
	}

	// Compilar reglas de deny
	for _, rule := range p.Deny {
		cr, err := compileRule(rule)
		if err != nil {
			return nil, fmt.Errorf("deny rule: %w", err)
		}
		compiled.DenyPatterns = append(compiled.DenyPatterns, cr)
	}

	// Compilar reglas de allow
	for _, rule := range p.Allow {
		cr, err := compileRule(rule)
		if err != nil {
			return nil, fmt.Errorf("allow rule: %w", err)
		}
		compiled.AllowPatterns = append(compiled.AllowPatterns, cr)
	}

	return compiled, nil
}

// compileRule convierte los patterns en regex
func compileRule(r Rule) (CompiledRule, error) {
	cr := CompiledRule{Original: r}

	if r.Model != "" {
		regex, err := patternToRegex(r.Model)
		if err != nil {
			return cr, fmt.Errorf("invalid model pattern %q: %w", r.Model, err)
		}
		cr.ModelRegex = regex
		cr.HasModel = true
	}

	if r.Provider != "" {
		regex, err := patternToRegex(r.Provider)
		if err != nil {
			return cr, fmt.Errorf("invalid provider pattern %q: %w", r.Provider, err)
		}
		cr.ProviderRegex = regex
		cr.HasProvider = true
	}

	return cr, nil
}

// patternToRegex convierte "gpt-*" a regex "^gpt-.*$"
func patternToRegex(pattern string) (*regexp.Regexp, error) {
	// Escapar caracteres especiales excepto *
	escaped := regexp.QuoteMeta(pattern)
	// Convertir \* a .*
	escaped = strings.ReplaceAll(escaped, `\*`, `.*`)
	// Anclar
	return regexp.Compile("^" + escaped + "$")
}

// Matches verifica si una regla matchea el contexto
func (cr *CompiledRule) Matches(model, provider string) bool {
	if cr.HasModel && !cr.ModelRegex.MatchString(model) {
		return false
	}
	if cr.HasProvider && !cr.ProviderRegex.MatchString(provider) {
		return false
	}
	return true
}
