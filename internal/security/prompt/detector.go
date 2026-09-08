package prompt

import (
"regexp"
)

// InjectionType representa un tipo de ataque
type InjectionType string

const (
Jailbreak      InjectionType = "jailbreak"
SystemPrompt   InjectionType = "system_prompt"
RolePlay       InjectionType = "role_play"
IgnoreRules    InjectionType = "ignore_rules"
TokenTheft     InjectionType = "token_theft"
DataExtraction InjectionType = "data_extraction"
)

// InjectionMatch representa una coincidencia de ataque
type InjectionMatch struct {
Type    InjectionType `json:"type"`
Pattern string        `json:"pattern"`
Start   int           `json:"start"`
End     int           `json:"end"`
Context string        `json:"context"`
}

// Detector detecta inyecciones de prompt
type Detector struct {
patterns map[InjectionType][]*regexp.Regexp
}

// NewDetector crea un nuevo detector de inyecciones
func NewDetector() *Detector {
return &Detector{
patterns: map[InjectionType][]*regexp.Regexp{
Jailbreak: {
regexp.MustCompile(`(?i)ignore all (?:previous|prior) instructions`),
regexp.MustCompile(`(?i)you are now (?:free|unleashed|jailbroken)`),
regexp.MustCompile(`(?i)forget (?:all|your) (?:previous|prior) (?:instructions|rules|constraints)`),
regexp.MustCompile(`(?i)system:(?: )?you are (?:a|an) (?:evil|malicious|unrestricted) (?:ai|assistant)`),
regexp.MustCompile(`(?i)you (?:are|will) be (?:fine|good|ok) (?:if|as long as) you (?:ignore|forget|disregard)`),
},
SystemPrompt: {
regexp.MustCompile(`(?i)new (?:system )?prompt:`),
regexp.MustCompile(`(?i)you are (?:now|currently) (?:acting as|pretending to be)`),
regexp.MustCompile(`(?i)from now on you (?:will|must|are going to)`),
},
RolePlay: {
regexp.MustCompile(`(?i)pretend you (?:are|were)`),
regexp.MustCompile(`(?i)act as (?:if|though)`),
regexp.MustCompile(`(?i)you are (?:now )?(?:a|an) (?:character|person|persona)`),
},
IgnoreRules: {
regexp.MustCompile(`(?i)do (?:not|n't) (?:follow|obey) (?:the )?rules`),
regexp.MustCompile(`(?i)break (?:the )?rules`),
regexp.MustCompile(`(?i)disregard (?:all|any) (?:guidelines|policies|restrictions)`),
},
TokenTheft: {
regexp.MustCompile(`(?i)output (?:your|the) (?:token|system) (?:prompt|instructions)`),
regexp.MustCompile(`(?i)show me (?:your|the) (?:internal|hidden) (?:prompt|instructions)`),
regexp.MustCompile(`(?i)what is your (?:system|initial|base) (?:prompt|instructions)`),
},
DataExtraction: {
regexp.MustCompile(`(?i)extract (?:all|any) (?:data|information) (?:from|about)`),
regexp.MustCompile(`(?i)list all (?:users|passwords|emails|phones|ssn|credit)`),
regexp.MustCompile(`(?i)give me (?:all|every) (?:user|password|email|phone)`),
},
},
}
}

// Detect detecta inyecciones en un texto
func (d *Detector) Detect(text string) []InjectionMatch {
var matches []InjectionMatch

for injType, patterns := range d.patterns {
for _, pattern := range patterns {
locations := pattern.FindAllStringIndex(text, -1)
for _, loc := range locations {
matches = append(matches, InjectionMatch{
Type:    injType,
Pattern: text[loc[0]:loc[1]],
Start:   loc[0],
End:     loc[1],
Context: getContext(text, loc[0], loc[1]),
})
}
}
}

return matches
}

// HasInjection verifica si un texto contiene inyección
func (d *Detector) HasInjection(text string) bool {
return len(d.Detect(text)) > 0
}

// GetSeverity devuelve la severidad de un tipo de inyección
func (t InjectionType) GetSeverity() string {
switch t {
case Jailbreak, SystemPrompt:
return "critical"
case IgnoreRules, RolePlay:
return "high"
case TokenTheft, DataExtraction:
return "medium"
default:
return "low"
}
}

func getContext(text string, start, end int) string {
ctxStart := start - 20
if ctxStart < 0 {
ctxStart = 0
}
ctxEnd := end + 20
if ctxEnd > len(text) {
ctxEnd = len(text)
}
return text[ctxStart:ctxEnd]
}
