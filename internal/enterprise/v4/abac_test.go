package enterprisev4

import "testing"

func TestABAC(t *testing.T) {
	e := Engine{Rules: []Rule{{Effect: "allow", Roles: []string{"admin"}, Environments: []string{"production"}, ResourceTypes: []string{"project"}}}}
	s := Subject{UserID: "u", OrgID: "o", Role: "admin", Environment: "production"}
	r := Resource{Type: "project", OrgID: "o", Environment: "production"}
	if !e.Allow(s, r) {
		t.Fatal("expected allow")
	}
	r.OrgID = "other"
	if e.Allow(s, r) {
		t.Fatal("cross org must deny")
	}
}
