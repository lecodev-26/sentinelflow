package config

import (
"os"
"time"

"gopkg.in/yaml.v3"
)

type Config struct {
Providers []Provider    `yaml:"providers"`
Cache     CacheConfig   `yaml:"cache"`
Rules     []Rule        `yaml:"rules"`
Metrics   MetricsConfig `yaml:"metrics"`
Logging   LoggingConfig `yaml:"logging"`
}

type Provider struct {
Name     string            `yaml:"name"`
URL      string            `yaml:"url"`
Timeout  time.Duration     `yaml:"timeout"`
Fallback string            `yaml:"fallback"`
Headers  map[string]string `yaml:"headers"`
}

type CacheConfig struct {
Enabled bool          `yaml:"enabled"`
TTL     time.Duration `yaml:"ttl"`
MaxSize int           `yaml:"max_size"`
}

type Rule struct {
Path      string   `yaml:"path"`
Method    string   `yaml:"method"`
Cache     bool     `yaml:"cache"`
Providers []string `yaml:"providers"`
}

type MetricsConfig struct {
Enabled bool   `yaml:"enabled"`
Port    int    `yaml:"port"`
Path    string `yaml:"path"`
}

type LoggingConfig struct {
Level  string `yaml:"level"`
Format string `yaml:"format"`
Output string `yaml:"output"`
}

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

func (c *Config) GetProviderByName(name string) *Provider {
for i := range c.Providers {
if c.Providers[i].Name == name {
return &c.Providers[i]
}
}
return nil
}

func (c *Config) GetRuleByPath(path, method string) *Rule {
for i := range c.Rules {
if c.Rules[i].Path == path && c.Rules[i].Method == method {
return &c.Rules[i]
}
}
return nil
}
