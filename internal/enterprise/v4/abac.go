package enterprisev4

type Subject struct {
	UserID, OrgID, Role, Environment string
	Attributes                       map[string]string
}
type Resource struct {
	Type, OrgID, ProjectID, Environment string
	Attributes                          map[string]string
}
type Rule struct {
	Effect        string
	Roles         []string
	Environments  []string
	ResourceTypes []string
}
type Engine struct{ Rules []Rule }

func (e Engine) Allow(s Subject, r Resource) bool {
	decision := false
	for _, x := range e.Rules {
		if !match(x.Roles, s.Role) || !match(x.Environments, s.Environment) || !match(x.ResourceTypes, r.Type) {
			continue
		}
		if x.Effect == "deny" {
			return false
		}
		if x.Effect == "allow" && s.OrgID == r.OrgID {
			if r.Environment == "" || r.Environment == s.Environment {
				decision = true
			}
		}
	}
	return decision
}
func match(xs []string, v string) bool {
	if len(xs) == 0 {
		return true
	}
	for _, x := range xs {
		if x == "*" || x == v {
			return true
		}
	}
	return false
}
