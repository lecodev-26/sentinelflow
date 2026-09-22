package cache

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"time"
)

// Embedder genera embeddings para texto
type Embedder interface {
	Embed(text string) ([]float64, error)
}

// L3Semantic es la caché semántica
type L3Semantic struct {
	mu        sync.RWMutex
	entries   map[string]*SemanticEntry
	embedder  Embedder
	threshold float64
}

// SemanticEntry es una entrada semántica
type SemanticEntry struct {
	Key       string    `json:"key"`
	Text      string    `json:"text"`
	Embedding []float64 `json:"embedding"`
	Value     *Value    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

// NewL3Semantic crea una caché semántica
func NewL3Semantic(embedder Embedder, threshold float64) *L3Semantic {
	if threshold <= 0 {
		threshold = 0.92
	}
	return &L3Semantic{
		entries:   make(map[string]*SemanticEntry),
		embedder:  embedder,
		threshold: threshold,
	}
}

func (l *L3Semantic) Name() string { return "L3-semantic" }

// GetByText busca por similitud semántica
func (l *L3Semantic) GetByText(ctx context.Context, text string) (*Value, bool) {
	if l.embedder == nil {
		return nil, false
	}

	embedding, err := l.embedder.Embed(text)
	if err != nil {
		return nil, false
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var best *Value
	var bestScore float64

	for _, e := range l.entries {
		if e.Value.IsExpired() {
			continue
		}
		score := cosineSimilarity(embedding, e.Embedding)
		if score >= l.threshold && score > bestScore {
			best = e.Value
			bestScore = score
		}
	}

	if best != nil {
		return best, true
	}
	return nil, false
}

// GetByKey implementa la interface Layer (no hace nada para semántica)
func (l *L3Semantic) Get(ctx context.Context, key Key) (*Value, bool, error) {
	return nil, false, nil
}

// SetByText guarda una entrada semántica
func (l *L3Semantic) SetByText(ctx context.Context, text string, value *Value, ttl time.Duration) error {
	if l.embedder == nil {
		return nil
	}

	embedding, err := l.embedder.Embed(text)
	if err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	key := value.Provider + ":" + value.Model + ":" + text
	l.entries[key] = &SemanticEntry{
		Key:       key,
		Text:      text,
		Embedding: embedding,
		Value:     value,
		CreatedAt: time.Now(),
	}
	return nil
}

func (l *L3Semantic) Set(ctx context.Context, key Key, value *Value, ttl time.Duration) error {
	return nil
}

func (l *L3Semantic) Delete(ctx context.Context, key Key) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key.String())
	return nil
}

func (l *L3Semantic) Clear(ctx context.Context, prefix string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k := range l.entries {
		if len(prefix) == 0 || (len(k) >= len(prefix) && k[:len(prefix)] == prefix) {
			delete(l.entries, k)
		}
	}
	return nil
}

// Size devuelve el número de entradas
func (l *L3Semantic) Size() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries)
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

var _ = json.Marshal
