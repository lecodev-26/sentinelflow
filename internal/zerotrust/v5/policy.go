package zerotrustv5

import "strings"

type Subject struct{ ID, Tenant, Project, Role string }
type Resource struct{ Tenant, Project, Type, ID string }
type Decision struct {
	Allowed bool
	Reason  string
}
type Policy struct{ AllowedRoles, AllowedTypes []string }

func (p Policy) Authorize(s Subject, r Resource) Decision {
	if s.ID == "" || s.Tenant == "" || r.Tenant == "" {
		return Decision{false, "missing identity or tenant"}
	}
	if s.Tenant != r.Tenant {
		return Decision{false, "tenant boundary"}
	}
	if r.Project != "" && s.Project != r.Project {
		return Decision{false, "project boundary"}
	}
	if len(p.AllowedRoles) > 0 && !contains(p.AllowedRoles, s.Role) {
		return Decision{false, "role denied"}
	}
	if len(p.AllowedTypes) > 0 && !contains(p.AllowedTypes, r.Type) {
		return Decision{false, "resource type denied"}
	}
	return Decision{true, "authorized"}
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if strings.EqualFold(v, x) {
			return true
		}
	}
	return false
}
