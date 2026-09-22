package v3

import (
	"context"
	"sync"
)

// EvalContext contiene la información para evaluar la política
type EvalContext struct {
	TenantID  string
	UserID    string
	ProjectID string
	Model     string
	Provider  string
	Stream    bool
	Tokens    int
	Metadata  map[string]interface{}
}

// EvalResult es el resultado de evaluar la política
type EvalResult struct {
	Allowed       bool            `json:"allowed"`
	Action        Action          `json:"action"`
	Reason        string          `json:"reason,omitempty"`
	PolicyID      string          `json:"policy_id"`
	PolicyVersion int             `json:"policy_version"`
	MatchedRule   string          `json:"matched_rule,omitempty"`
	Limits        *Limits         `json:"limits,omitempty"`
	Routing       *RoutingPolicy  `json:"routing,omitempty"`
	Security      *SecurityPolicy `json:"security,omitempty"`
}

// Evaluator evalúa políticas compiladas
type Evaluator struct {
	mu       sync.RWMutex
	policies map[string][]*CompiledPolicy // tenantID → policies (ordenadas por priority)
	global   []*CompiledPolicy
}

// NewEvaluator crea un evaluador
func NewEvaluator() *Evaluator {
	return &Evaluator{
		policies: make(map[string][]*CompiledPolicy),
	}
}

// Register añade una política compilada
func (e *Evaluator) Register(compiled *CompiledPolicy) {
	e.mu.Lock()
	defer e.mu.Unlock()

	tenantID := compiled.Source.TenantID
	if tenantID == "" {
		// Política global
		e.global = append(e.global, compiled)
		sortByPriority(e.global)
	} else {
		e.policies[tenantID] = append(e.policies[tenantID], compiled)
		sortByPriority(e.policies[tenantID])
	}
}

// Evaluate evalúa las políticas aplicables
func (e *Evaluator) Evaluate(ctx context.Context, evalCtx *EvalContext) *EvalResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := &EvalResult{
		Allowed: true,
		Action:  ActionAllow,
	}

	// Aplicar políticas del tenant primero, luego globales
	var toEvaluate []*CompiledPolicy
	if tenantPolicies, ok := e.policies[evalCtx.TenantID]; ok {
		toEvaluate = append(toEvaluate, tenantPolicies...)
	}
	toEvaluate = append(toEvaluate, e.global...)

	for _, policy := range toEvaluate {
		if !policy.Source.Enabled {
			continue
		}

		// Verificar reglas de deny
		for _, rule := range policy.DenyPatterns {
			if rule.Matches(evalCtx.Model, evalCtx.Provider) {
				return &EvalResult{
					Allowed:       false,
					Action:        ActionDeny,
					Reason:        rule.Original.Reason,
					PolicyID:      policy.Source.ID,
					PolicyVersion: policy.Source.Version,
					MatchedRule:   "deny:" + rule.Original.Model + "|" + rule.Original.Provider,
				}
			}
		}

		// Aplicar limits
		if policy.Source.Limits != nil {
			result.Limits = policy.Source.Limits
		}

		// Aplicar routing policy
		if policy.Source.Routing != nil {
			result.Routing = policy.Source.Routing
		}

		// Aplicar security policy
		if policy.Source.Security != nil {
			result.Security = policy.Source.Security
		}

		result.PolicyID = policy.Source.ID
		result.PolicyVersion = policy.Source.Version
	}

	return result
}

// List devuelve todas las políticas registradas
func (e *Evaluator) List() []*CompiledPolicy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := []*CompiledPolicy{}
	for _, policies := range e.policies {
		result = append(result, policies...)
	}
	result = append(result, e.global...)
	return result
}

func sortByPriority(policies []*CompiledPolicy) {
	for i := 0; i < len(policies); i++ {
		for j := i + 1; j < len(policies); j++ {
			if policies[j].Source.Priority < policies[i].Source.Priority {
				policies[i], policies[j] = policies[j], policies[i]
			}
		}
	}
}
