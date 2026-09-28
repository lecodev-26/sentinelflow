package pluginsv5

import "testing"

func TestPluginRegistry(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(Plugin{ID: "p", Version: "1", Type: Provider})
	if _, ok := r.Get("p"); !ok {
		t.Fatal("missing plugin")
	}
}
