package ssrf

import "testing"

func TestSSRFBlocking(t *testing.T) {
	v := NewValidator()

	badURLs := []string{
		"http://localhost:8080",
		"http://127.0.0.1:8080",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1",
		"http://192.168.1.1",
		"file:///etc/passwd",
		"gopher://example.com",
	}

	for _, u := range badURLs {
		if err := v.Validate(u); err == nil {
			t.Errorf("Expected SSRF block for %s", u)
		}
	}
}
