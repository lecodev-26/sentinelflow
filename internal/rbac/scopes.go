package rbac

// Scope representa un permiso granular sobre un recurso
type Scope string

const (
	// Scopes de lectura
	ScopeReadChat      Scope = "chat:read"
	ScopeReadModels    Scope = "models:read"
	ScopeReadUsage     Scope = "usage:read"
	ScopeReadProviders Scope = "providers:read"
	ScopeReadAdmin     Scope = "admin:read"
	ScopeReadTraces    Scope = "traces:read"
	ScopeReadMetrics   Scope = "metrics:read"
	ScopeReadAudit     Scope = "audit:read"
	ScopeReadAnalytics Scope = "analytics:read"

	// Scopes de escritura
	ScopeWriteChat      Scope = "chat:write"
	ScopeWriteAdmin     Scope = "admin:write"
	ScopeWriteProviders Scope = "providers:write"
	ScopeWriteUsers     Scope = "users:write"
	ScopeWriteKeys      Scope = "keys:write"

	// Scopes de gestión
	ScopeManageOrg     Scope = "org:manage"
	ScopeManageTenant  Scope = "tenant:manage"
	ScopeManageBilling Scope = "billing:manage"
	ScopeManagePolicy  Scope = "policy:manage"

	// Wildcard
	ScopeAll Scope = "*"
)

// ScopeSet es un conjunto de scopes
type ScopeSet map[Scope]bool

// NewScopeSet crea un conjunto de scopes
func NewScopeSet(scopes ...Scope) ScopeSet {
	set := make(ScopeSet)
	for _, s := range scopes {
		set[s] = true
	}
	return set
}

// Has verifica si el conjunto contiene un scope
func (s ScopeSet) Has(scope Scope) bool {
	if s[ScopeAll] {
		return true
	}
	return s[scope]
}

// HasAny verifica si el conjunto tiene alguno de los scopes
func (s ScopeSet) HasAny(scopes ...Scope) bool {
	for _, scope := range scopes {
		if s.Has(scope) {
			return true
		}
	}
	return false
}

// HasAll verifica si el conjunto tiene todos los scopes
func (s ScopeSet) HasAll(scopes ...Scope) bool {
	for _, scope := range scopes {
		if !s.Has(scope) {
			return false
		}
	}
	return true
}

// Add añade un scope
func (s ScopeSet) Add(scope Scope) {
	s[scope] = true
}

// Remove elimina un scope
func (s ScopeSet) Remove(scope Scope) {
	delete(s, scope)
}

// List devuelve todos los scopes como slice
func (s ScopeSet) List() []Scope {
	result := make([]Scope, 0, len(s))
	for scope := range s {
		result = append(result, scope)
	}
	return result
}

// RoleScopes define los scopes por defecto de cada rol
var RoleScopes = map[Role][]Scope{
	RoleAdmin: {
		ScopeAll,
	},
	RoleEditor: {
		ScopeReadChat, ScopeWriteChat,
		ScopeReadModels,
		ScopeReadUsage,
		ScopeReadProviders,
		ScopeReadAdmin,
		ScopeReadTraces,
		ScopeReadMetrics,
	},
	RoleViewer: {
		ScopeReadChat,
		ScopeReadModels,
		ScopeReadUsage,
		ScopeReadProviders,
		ScopeReadAdmin,
		ScopeReadTraces,
		ScopeReadMetrics,
	},
	RoleMember: {
		ScopeReadChat, ScopeWriteChat,
		ScopeReadModels,
		ScopeReadUsage,
	},
}

// ScopesForRole devuelve los scopes de un rol
func ScopesForRole(role Role) ScopeSet {
	scopes, exists := RoleScopes[role]
	if !exists {
		return NewScopeSet()
	}
	return NewScopeSet(scopes...)
}
