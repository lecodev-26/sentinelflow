package controlplanev4

import "testing"

func TestPolicyLifecycle(t *testing.T) {
	m := NewPolicyManager()
	p, e := m.Create("p", map[string]any{"allow": true})
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Validate("p", p.Version); e != nil {
		t.Fatal(e)
	}
	if e = m.Publish("p", p.Version, 100); e != nil {
		t.Fatal(e)
	}
	if _, ok := m.Active("p"); !ok {
		t.Fatal("not active")
	}
	if e = m.Rollback("p"); e != nil {
		t.Fatal(e)
	}
}
