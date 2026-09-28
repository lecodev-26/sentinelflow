package identity

import "testing"

func TestIsValidEnvironment(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty uses production default", value: "", want: true},
		{name: "development", value: "development", want: true},
		{name: "staging", value: "staging", want: true},
		{name: "production", value: "production", want: true},
		{name: "prod alias rejected", value: "prod", want: false},
		{name: "invalid rejected", value: "sandbox", want: false},
		{name: "case sensitive", value: "Production", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidEnvironment(tt.value); got != tt.want {
				t.Fatalf("isValidEnvironment(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
