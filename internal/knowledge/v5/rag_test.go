package knowledgev5

import (
	"context"
	"testing"
)

func TestIndexTenantProjectIsolation(t *testing.T) {
	i := NewIndex()
	_ = i.Ingest(context.Background(), Document{ID: "d", TenantID: "a", ProjectID: "p", Text: "private architecture"}, 100)
	if len(i.Search(context.Background(), "b", "p", "architecture", 5)) != 0 {
		t.Fatal("tenant leak")
	}
	if len(i.Search(context.Background(), "a", "p", "architecture", 5)) == 0 {
		t.Fatal("missing hit")
	}
}
