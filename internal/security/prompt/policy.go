package prompt

// Action define qué hacer cuando se detecta una inyección
type Action string

const (
ActionBlock Action = "block" // Bloquear la petición
ActionWarn  Action = "warn"  // Registrar pero permitir
ActionAllow Action = "allow" // Permitir sin acción
)

// Policy define cómo manejar diferentes tipos de inyección
type Policy struct {
Actions map[InjectionType]Action `json:"actions"`
BlockOnCritical bool             `json:"block_on_critical"`
}

// DefaultPolicy devuelve la política por defecto
func DefaultPolicy() Policy {
return Policy{
Actions: map[InjectionType]Action{
Jailbreak:      ActionBlock,
SystemPrompt:   ActionBlock,
RolePlay:       ActionWarn,
IgnoreRules:    ActionBlock,
TokenTheft:     ActionBlock,
DataExtraction: ActionWarn,
},
BlockOnCritical: true,
}
}

// GetAction devuelve la acción para un tipo de inyección
func (p *Policy) GetAction(injType InjectionType) Action {
if action, exists := p.Actions[injType]; exists {
return action
}
return ActionAllow
}

// Process procesa un texto según la política
func (p *Policy) Process(text string, detector *Detector) (string, []InjectionMatch, Action) {
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
if action == ActionWarn && maxAction != ActionBlock {
maxAction = ActionWarn
}
}

return text, matches, maxAction
}
