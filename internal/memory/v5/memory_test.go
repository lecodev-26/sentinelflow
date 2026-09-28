package memoryv5

import (
	"context"
	"testing"
)

func TestTenantIsolation(t *testing.T) {
	s := NewStore()
	_ = s.Put(context.Background(), Item{ID: "1", Scope: Scope{TenantID: "a"}, Text: "secret project"})
	if _, ok := s.Get(context.Background(), Scope{TenantID: "b"}, "1"); ok {
		t.Fatal("cross tenant memory leak")
	}
}
func TestSearch(t *testing.T) {
	s := NewStore()
	_ = s.Put(context.Background(), Item{ID: "1", Scope: Scope{TenantID: "a"}, Text: "sentinelflow routing policy"})
	if len(s.Search(context.Background(), Scope{TenantID: "a"}, "routing", 5)) != 1 {
		t.Fatal("not found")
	}
}
