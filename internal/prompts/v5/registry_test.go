package promptsv5

import "testing"

func TestPromptLifecycle(t *testing.T) {
	r := NewRegistry()
	v, _ := r.Create("support", "hello {{name}}")
	if err := r.Publish("support", v.Number); err != nil {
		t.Fatal(err)
	}
	if a, ok := r.Active("support"); !ok || a.Template != "hello {{name}}" {
		t.Fatal(a, ok)
	}
}
