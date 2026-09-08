package policy

// DefaultPolicies devuelve políticas predefinidas
func DefaultPolicies() []Policy {
return []Policy{
{
ID:          "pol_default_allow",
Name:        "Default Allow",
Description: "Permitir todas las peticiones por defecto",
Priority:    100,
Enabled:     true,
Conditions: []Condition{
{Type: "always", Field: "tenant_id", Op: "eq", Value: "*"},
},
Actions: []Action{
{Type: "allow", Params: map[string]interface{}{}},
},
},
{
ID:          "pol_block_openai_failures",
Name:        "Block OpenAI Failures",
Description: "Bloquear peticiones a OpenAI si hay muchos fallos",
Priority:    50,
Enabled:     true,
Conditions: []Condition{
{Type: "provider", Field: "provider", Op: "eq", Value: "openai"},
{Type: "status", Field: "status", Op: "gte", Value: 500},
},
Actions: []Action{
{Type: "deny", Params: map[string]interface{}{
"reason": "OpenAI provider failure",
}},
{Type: "alert", Params: map[string]interface{}{
"severity": "critical",
}},
},
},
{
ID:          "pol_cost_limit",
Name:        "Cost Limit",
Description: "Limitar coste por petición",
Priority:    40,
Enabled:     true,
Conditions: []Condition{
{Type: "cost", Field: "estimated_cost", Op: "gt", Value: 0.05},
},
Actions: []Action{
{Type: "deny", Params: map[string]interface{}{
"reason": "Estimated cost exceeds limit",
}},
{Type: "log", Params: map[string]interface{}{
"level": "warn",
}},
},
},
{
ID:          "pol_block_pii",
Name:        "Block PII",
Description: "Bloquear peticiones que contienen PII",
Priority:    30,
Enabled:     true,
Conditions: []Condition{
{Type: "pii", Field: "input", Op: "contains", Value: "email"},
},
Actions: []Action{
{Type: "redact", Params: map[string]interface{}{
"type": "pii",
}},
{Type: "log", Params: map[string]interface{}{
"level": "warn",
}},
},
},
{
ID:          "pol_block_prompt_injection",
Name:        "Block Prompt Injection",
Description: "Bloquear intentos de inyección de prompts",
Priority:    20,
Enabled:     true,
Conditions: []Condition{
{Type: "prompt_injection", Field: "input", Op: "eq", Value: true},
},
Actions: []Action{
{Type: "deny", Params: map[string]interface{}{
"reason": "Prompt injection detected",
}},
{Type: "alert", Params: map[string]interface{}{
"severity": "critical",
}},
{Type: "log", Params: map[string]interface{}{
"level": "block",
}},
},
},
}
}
