package zerotrustv5

import "testing"

func TestZeroTrustTenantBoundary(t *testing.T) {
	p := Policy{}
	d := p.Authorize(Subject{ID: "u", Tenant: "a"}, Resource{Tenant: "b"})
	if d.Allowed {
		t.Fatal(d)
	}
}
