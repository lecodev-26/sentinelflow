package config

import (
"os"
"testing"
"time"
)

func TestLoadConfig(t *testing.T) {
// Crear un archivo de configuración temporal
content := `
providers:
  - name: test-provider
    url: "https://test.com"
    timeout: 10s
    fallback: ""
    headers: {}

cache:
  enabled: true
  ttl: 60s
  max_size: 100

rules:
  - path: "/test"
    method: "POST"
    cache: true
    providers: ["test-provider"]

metrics:
  enabled: true
  port: 9090
  path: "/metrics"

logging:
  level: "info"
  format: "json"
  output: "stdout"
`

tmpfile, err := os.CreateTemp("", "config-*.yaml")
if err != nil {
t.Fatal(err)
}
defer os.Remove(tmpfile.Name())

if _, err := tmpfile.Write([]byte(content)); err != nil {
t.Fatal(err)
}
if err := tmpfile.Close(); err != nil {
t.Fatal(err)
}

// Cargar la configuración
cfg, err := LoadConfig(tmpfile.Name())
if err != nil {
t.Fatalf("LoadConfig failed: %v", err)
}

// Verificar proveedores
if len(cfg.Providers) != 1 {
t.Errorf("Expected 1 provider, got %d", len(cfg.Providers))
}
if cfg.Providers[0].Name != "test-provider" {
t.Errorf("Expected provider name 'test-provider', got '%s'", cfg.Providers[0].Name)
}
if cfg.Providers[0].URL != "https://test.com" {
t.Errorf("Expected URL 'https://test.com', got '%s'", cfg.Providers[0].URL)
}

// Verificar caché
if !cfg.Cache.Enabled {
t.Error("Expected cache enabled")
}
if cfg.Cache.TTL != 60*time.Second {
t.Errorf("Expected TTL 60s, got %v", cfg.Cache.TTL)
}

// Verificar reglas
if len(cfg.Rules) != 1 {
t.Errorf("Expected 1 rule, got %d", len(cfg.Rules))
}
if cfg.Rules[0].Path != "/test" {
t.Errorf("Expected path '/test', got '%s'", cfg.Rules[0].Path)
}

// Verificar métricas
if !cfg.Metrics.Enabled {
t.Error("Expected metrics enabled")
}
if cfg.Metrics.Port != 9090 {
t.Errorf("Expected port 9090, got %d", cfg.Metrics.Port)
}

// Verificar logging
if cfg.Logging.Level != "info" {
t.Errorf("Expected level 'info', got '%s'", cfg.Logging.Level)
}
if cfg.Logging.Format != "json" {
t.Errorf("Expected format 'json', got '%s'", cfg.Logging.Format)
}
}
