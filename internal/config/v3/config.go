package v3

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config es la configuración completa de SentinelFlow V3
type Config struct {
	Static  StaticConfig  `json:"static"`
	Dynamic DynamicConfig `json:"dynamic"`
	Secrets SecretsConfig `json:"secrets"`
}

// StaticConfig viene de variables de entorno + flags (no cambia en runtime)
type StaticConfig struct {
	Environment  string        `json:"environment"`  // development, staging, production
	GatewayPort  int           `json:"gateway_port"` // 8080
	AdminPort    int           `json:"admin_port"`   // 8081
	MetricsPort  int           `json:"metrics_port"` // 9090
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
	LogLevel     string        `json:"log_level"`
	LogFormat    string        `json:"log_format"` // json, text
}

// DynamicConfig viene de PostgreSQL (Control Plane)
// Se recarga sin reiniciar el gateway
type DynamicConfig struct {
	Version   int64                  `json:"version"`
	Providers map[string]interface{} `json:"providers"`
	Routing   map[string]interface{} `json:"routing"`
	Policies  []interface{}          `json:"policies"`
}

// SecretsConfig viene de env vars o secret manager
type SecretsConfig struct {
	DatabaseURL  string `json:"database_url"`
	RedisURL     string `json:"redis_url,omitempty"`
	VaultKey     string `json:"vault_key,omitempty"`
	OpenAIKey    string `json:"openai_key,omitempty"`
	AnthropicKey string `json:"anthropic_key,omitempty"`
}

// Load carga toda la configuración
func Load() (*Config, error) {
	cfg := &Config{
		Static:  loadStatic(),
		Dynamic: DynamicConfig{}, // se rellena desde PostgreSQL
		Secrets: loadSecrets(),
	}

	// Validar
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate valida la configuración
func (c *Config) Validate() error {
	// En producción, DATABASE_URL es obligatoria
	if c.Static.Environment == "production" && c.Secrets.DatabaseURL == "" {
		return fmt.Errorf("SENTINELFLOW_DATABASE_URL is required in production")
	}

	// En producción, VAULT_KEY es obligatoria
	if c.Static.Environment == "production" && c.Secrets.VaultKey == "" {
		return fmt.Errorf("SENTINELFLOW_VAULT_KEY is required in production")
	}

	// Puertos válidos
	if c.Static.GatewayPort <= 0 || c.Static.GatewayPort > 65535 {
		return fmt.Errorf("invalid gateway_port: %d", c.Static.GatewayPort)
	}

	return nil
}

// IsProduction devuelve true si estamos en producción
func (c *Config) IsProduction() bool {
	return c.Static.Environment == "production"
}

// IsDevelopment devuelve true si estamos en desarrollo
func (c *Config) IsDevelopment() bool {
	return c.Static.Environment == "development" || c.Static.Environment == ""
}

// loadStatic carga la configuración estática
func loadStatic() StaticConfig {
	return StaticConfig{
		Environment:  getEnv("SENTINELFLOW_ENV", "development"),
		GatewayPort:  getEnvInt("SENTINELFLOW_GATEWAY_PORT", 8080),
		AdminPort:    getEnvInt("SENTINELFLOW_ADMIN_PORT", 8081),
		MetricsPort:  getEnvInt("SENTINELFLOW_METRICS_PORT", 9090),
		ReadTimeout:  getEnvDuration("SENTINELFLOW_READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getEnvDuration("SENTINELFLOW_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:  getEnvDuration("SENTINELFLOW_IDLE_TIMEOUT", 60*time.Second),
		LogLevel:     getEnv("SENTINELFLOW_LOG_LEVEL", "info"),
		LogFormat:    getEnv("SENTINELFLOW_LOG_FORMAT", "json"),
	}
}

// loadSecrets carga los secretos
func loadSecrets() SecretsConfig {
	return SecretsConfig{
		DatabaseURL:  getEnv("SENTINELFLOW_DATABASE_URL", ""),
		RedisURL:     getEnv("REDIS_URL", ""),
		VaultKey:     getEnv("SENTINELFLOW_VAULT_KEY", ""),
		OpenAIKey:    getEnv("OPENAI_API_KEY", ""),
		AnthropicKey: getEnv("ANTHROPIC_API_KEY", ""),
	}
}

// Helpers

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
