package pii

// Action define qué hacer cuando se detecta PII
type Action string

const (
ActionRedact Action = "redact" // Ocultar información
ActionBlock  Action = "block"  // Bloquear la petición
ActionWarn   Action = "warn"   // Registrar pero permitir
ActionAllow  Action = "allow"  // Permitir sin acción
)

// Policy define cómo manejar diferentes tipos de PII
type Policy struct {
Actions map[PIIType]Action `json:"actions"`
}

// DefaultPolicy devuelve la política por defecto
func DefaultPolicy() Policy {
return Policy{
Actions: map[PIIType]Action{
Email:      ActionRedact,
Phone:      ActionRedact,
CreditCard: ActionBlock,
SSN:        ActionBlock,
IPAddress:  ActionWarn,
URL:        ActionAllow,
},
}
}

// GetAction devuelve la acción para un tipo de PII
func (p *Policy) GetAction(piiType PIIType) Action {
if action, exists := p.Actions[piiType]; exists {
return action
}
return ActionAllow
}

// Process procesa un texto según la política
func (p *Policy) Process(text string, detector *Detector) (string, []PIIMatch, Action) {
matches := detector.Detect(text)
if len(matches) == 0 {
return text, nil, ActionAllow
}

// Verificar la acción más estricta
maxAction := ActionAllow
for _, match := range matches {
action := p.GetAction(match.Type)
if action == ActionBlock {
return text, matches, ActionBlock
}
if action == ActionRedact && maxAction != ActionBlock {
maxAction = ActionRedact
}
if action == ActionWarn && maxAction != ActionBlock && maxAction != ActionRedact {
maxAction = ActionWarn
}
}

// Si la acción es redact, redactar el texto
if maxAction == ActionRedact {
return detector.Redact(text), matches, ActionRedact
}

return text, matches, maxAction
}
