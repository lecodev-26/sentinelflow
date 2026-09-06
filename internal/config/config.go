package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config representa toda la configuración del proxy
type Config struct {
	Providers []Provider    `yaml:"providers"`
	Cache     CacheConfig   `yaml:"cache"`
	Rules     []Rule        `yaml:"rules"`
	Metrics   MetricsConfig `yaml:"metrics"`
	Logging   LoggingConfig `yaml:"logging"`
}

// Provider define un proveedor de IA (OpenAI, Anthropic, etc.)
type Provider struct {
	Name     string            `yaml:"name"`
	URL      string            `yaml:"url"`
	Timeout  time.Duration     `yaml:"timeout"`
	Fallback string            `yaml:"fallback"`
	Headers  map[string]string `yaml:"headers"`
}

// CacheConfig configuración de caché
type CacheConfig struct {
	Enabled bool          `yaml:"enabled"`
	TTL     time.Duration `yaml:"ttl"`
	MaxSize int           `yaml:"max_size"`
}

// Rule define qué rutas se enrutan a qué proveedores
type Rule struct {
	Path      string   `yaml:"path"`
	Method    string   `yaml:"method"`
	Cache     bool     `yaml:"cache"`
	Providers []string `yaml:"providers"`
}

// MetricsConfig configuración de métricas
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// LoggingConfig configuración de logs
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// LoadConfig carga y parsea el archivo de configuración
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// GetProviderByName busca un proveedor por nombre
func (c *Config) GetProviderByName(name string) *Provider {
	for i := range c.Providers {
		if c.Providers[i].Name == name {
			return &c.Providers[i]
		}
	}
	return nil
}

// GetRuleByPath busca una regla por path y método
func (c *Config) GetRuleByPath(path, method string) *Rule {
	for i := range c.Rules {
		if c.Rules[i].Path == path && c.Rules[i].Method == method {
			return &c.Rules[i]
		}
	}
	return nil
}
