package semantic

import (
"math"
"sync"
"time"
)

// Entry representa una entrada en la caché semántica
type Entry struct {
Prompt     string    `json:"prompt"`
Response   []byte    `json:"response"`
Embedding  []float64 `json:"embedding"`
Timestamp  time.Time `json:"timestamp"`
Model      string    `json:"model"`
Provider   string    `json:"provider"`
TTL        time.Duration `json:"ttl"`
}

// Cache es una caché semántica
type Cache struct {
mu        sync.RWMutex
entries   map[string]*Entry
embedder  Embedder
threshold float64
ttl       time.Duration
maxSize   int
}

// Config configuración de la caché semántica
type Config struct {
Threshold float64
TTL       time.Duration
MaxSize   int
}

// DefaultConfig devuelve la configuración por defecto
func DefaultConfig() Config {
return Config{
Threshold: 0.85,
TTL:       10 * time.Minute,
MaxSize:   1000,
}
}

// NewCache crea una nueva caché semántica
func NewCache(embedder Embedder, config Config) *Cache {
return &Cache{
entries:   make(map[string]*Entry),
embedder:  embedder,
threshold: config.Threshold,
ttl:       config.TTL,
maxSize:   config.MaxSize,
}
}

// Get busca una respuesta en caché por similitud semántica
func (c *Cache) Get(prompt string, model string) ([]byte, bool) {
c.mu.RLock()
defer c.mu.RUnlock()

// Generar embedding para el prompt
embedding, err := c.embedder.Embed(prompt)
if err != nil {
return nil, false
}

var bestMatch []byte
var bestScore float64

for _, entry := range c.entries {
// Verificar expiración
if time.Since(entry.Timestamp) > entry.TTL {
continue
}

// Verificar que el modelo coincida
if entry.Model != model {
continue
}

// Calcular similitud
score := cosineSimilarity(embedding, entry.Embedding)
if score > c.threshold && score > bestScore {
bestScore = score
bestMatch = entry.Response
}
}

if bestMatch != nil {
return bestMatch, true
}
return nil, false
}

// Set guarda una respuesta en caché
func (c *Cache) Set(prompt string, response []byte, model, provider string) error {
embedding, err := c.embedder.Embed(prompt)
if err != nil {
return err
}

c.mu.Lock()
defer c.mu.Unlock()

// Si la caché está llena, eliminar la entrada más antigua
if len(c.entries) >= c.maxSize {
var oldest string
var oldestTime time.Time
for key, entry := range c.entries {
if oldest == "" || entry.Timestamp.Before(oldestTime) {
oldest = key
oldestTime = entry.Timestamp
}
}
if oldest != "" {
delete(c.entries, oldest)
}
}

c.entries[prompt] = &Entry{
Prompt:    prompt,
Response:  response,
Embedding: embedding,
Timestamp: time.Now(),
Model:     model,
Provider:  provider,
TTL:       c.ttl,
}

return nil
}

// Delete elimina una entrada de la caché
func (c *Cache) Delete(prompt string) {
c.mu.Lock()
defer c.mu.Unlock()
delete(c.entries, prompt)
}

// Clear vacía la caché
func (c *Cache) Clear() {
c.mu.Lock()
defer c.mu.Unlock()
c.entries = make(map[string]*Entry)
}

// Size devuelve el número de entradas
func (c *Cache) Size() int {
c.mu.RLock()
defer c.mu.RUnlock()
return len(c.entries)
}

// CleanExpired elimina entradas expiradas
func (c *Cache) CleanExpired() {
c.mu.Lock()
defer c.mu.Unlock()

for key, entry := range c.entries {
if time.Since(entry.Timestamp) > entry.TTL {
delete(c.entries, key)
}
}
}

// cosineSimilarity calcula la similitud coseno entre dos vectores
func cosineSimilarity(a, b []float64) float64 {
if len(a) != len(b) || len(a) == 0 {
return 0
}

var dot, normA, normB float64
for i := 0; i < len(a); i++ {
dot += a[i] * b[i]
normA += a[i] * a[i]
normB += b[i] * b[i]
}

if normA == 0 || normB == 0 {
return 0
}

return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
