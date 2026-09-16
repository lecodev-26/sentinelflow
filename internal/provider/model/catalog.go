package model

// DefaultCatalog devuelve el catálogo por defecto con modelos conocidos
func DefaultCatalog() []*Model {
return []*Model{
// === OpenAI ===
{
ID:       "gpt-3.5-turbo",
Name:     "GPT-3.5 Turbo",
Provider: "openai",
Family:   "gpt-3.5",
Capabilities: []Capability{
CapabilityChat, CapabilityStream, CapabilityTools, CapabilityJSON, CapabilityFunction,
},
Pricing: Pricing{
InputPer1M:  0.50,
OutputPer1M: 1.50,
},
Limits: Limits{
MaxContextTokens: 16385,
MaxOutputTokens:  4096,
},
},
{
ID:       "gpt-4",
Name:     "GPT-4",
Provider: "openai",
Family:   "gpt-4",
Capabilities: []Capability{
CapabilityChat, CapabilityStream, CapabilityTools, CapabilityVision, CapabilityJSON, CapabilityFunction,
},
Pricing: Pricing{
InputPer1M:  30.00,
OutputPer1M: 60.00,
},
Limits: Limits{
MaxContextTokens: 8192,
MaxOutputTokens:  4096,
},
},
{
ID:       "gpt-4-turbo",
Name:     "GPT-4 Turbo",
Provider: "openai",
Family:   "gpt-4",
Capabilities: []Capability{
CapabilityChat, CapabilityStream, CapabilityTools, CapabilityVision, CapabilityJSON, CapabilityFunction,
},
Pricing: Pricing{
InputPer1M:  10.00,
OutputPer1M: 30.00,
},
Limits: Limits{
MaxContextTokens: 128000,
MaxOutputTokens:  4096,
},
},

// === Anthropic ===
{
ID:       "claude-3",
Name:     "Claude 3",
Provider: "anthropic",
Family:   "claude-3",
Capabilities: []Capability{
CapabilityChat, CapabilityStream, CapabilityVision, CapabilityTools,
},
Pricing: Pricing{
InputPer1M:  3.00,
OutputPer1M: 15.00,
},
Limits: Limits{
MaxContextTokens: 200000,
MaxOutputTokens:  4096,
},
},
{
ID:       "claude-3-sonnet",
Name:     "Claude 3 Sonnet",
Provider: "anthropic",
Family:   "claude-3",
Capabilities: []Capability{
CapabilityChat, CapabilityStream, CapabilityVision, CapabilityTools,
},
Pricing: Pricing{
InputPer1M:  3.00,
OutputPer1M: 15.00,
},
Limits: Limits{
MaxContextTokens: 200000,
MaxOutputTokens:  4096,
},
},
{
ID:       "claude-3-opus",
Name:     "Claude 3 Opus",
Provider: "anthropic",
Family:   "claude-3",
Capabilities: []Capability{
CapabilityChat, CapabilityStream, CapabilityVision, CapabilityTools,
},
Pricing: Pricing{
InputPer1M:  15.00,
OutputPer1M: 75.00,
},
Limits: Limits{
MaxContextTokens: 200000,
MaxOutputTokens:  4096,
},
},

// === Local ===
{
ID:       "llama3",
Name:     "Llama 3",
Provider: "local-llama",
Family:   "llama",
Capabilities: []Capability{
CapabilityChat, CapabilityStream,
},
Pricing: Pricing{
InputPer1M:  0.0,
OutputPer1M: 0.0,
},
Limits: Limits{
MaxContextTokens: 8192,
MaxOutputTokens:  4096,
},
},
{
ID:       "llama3.1",
Name:     "Llama 3.1",
Provider: "local-llama",
Family:   "llama",
Capabilities: []Capability{
CapabilityChat, CapabilityStream, CapabilityTools,
},
Pricing: Pricing{
InputPer1M:  0.0,
OutputPer1M: 0.0,
},
Limits: Limits{
MaxContextTokens: 128000,
MaxOutputTokens:  4096,
},
},
}
}

// DefaultCapabilities devuelve las capacidades que un provider debe declarar
func DefaultCapabilities(provider string) []Capability {
switch provider {
case "openai":
return []Capability{CapabilityChat, CapabilityStream, CapabilityTools, CapabilityVision, CapabilityJSON, CapabilityFunction}
case "anthropic":
return []Capability{CapabilityChat, CapabilityStream, CapabilityTools, CapabilityVision}
case "local-llama":
return []Capability{CapabilityChat, CapabilityStream}
default:
return []Capability{CapabilityChat, CapabilityStream}
}
}
