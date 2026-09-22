package v3

import (
	"context"
)

// Scanner detecta un tipo específico de amenaza
type Scanner interface {
	Name() string
	Scan(ctx context.Context, text string) []Finding
}

// Pipeline ejecuta todos los scanners
type Pipeline struct {
	scanners []Scanner
}

// NewPipeline crea un pipeline con scanners por defecto
func NewPipeline() *Pipeline {
	return &Pipeline{
		scanners: []Scanner{
			NewPIIScanner(),
			NewSecretScanner(),
			NewPromptInjectionScanner(),
			NewSSRFScanner(),
		},
	}
}

// Register añade un scanner
func (p *Pipeline) Register(s Scanner) {
	p.scanners = append(p.scanners, s)
}

// Evaluate evalúa un texto con todos los scanners
func (p *Pipeline) Evaluate(ctx context.Context, text string) *Decision {
	decision := NewDecision()

	for _, scanner := range p.scanners {
		findings := scanner.Scan(ctx, text)
		for _, f := range findings {
			decision.AddFinding(f)
		}
	}

	// Si hay redacción, generar texto procesado
	if decision.Action == ActionRedact && len(decision.Findings) > 0 {
		decision.ProcessedText = redactText(text, decision.Findings)
	}

	return decision
}

// redactText redacta los matches en el texto
func redactText(text string, findings []Finding) string {
	// Ordenar de derecha a izquierda para no alterar índices
	type idx struct{ start, end int }
	var indexes []idx
	for _, f := range findings {
		if f.Start < f.End && f.End <= len(text) {
			indexes = append(indexes, idx{f.Start, f.End})
		}
	}

	// Ordenar por start descendente
	for i := 0; i < len(indexes); i++ {
		for j := i + 1; j < len(indexes); j++ {
			if indexes[j].start > indexes[i].start {
				indexes[i], indexes[j] = indexes[j], indexes[i]
			}
		}
	}

	result := []byte(text)
	for _, ix := range indexes {
		if ix.end > len(result) {
			continue
		}
		redacted := make([]byte, ix.end-ix.start)
		for i := range redacted {
			redacted[i] = '*'
		}
		result = append(result[:ix.start], append(redacted, result[ix.end:]...)...)
	}

	return string(result)
}
