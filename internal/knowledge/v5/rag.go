package knowledgev5

import (
	"context"
	"errors"
	"sort"
	"sync"
)

type Document struct {
	ID, Title, Text     string
	TenantID, ProjectID string
	Metadata            map[string]string
}
type Chunk struct {
	ID, DocumentID, Text string
	TenantID, ProjectID  string
	Metadata             map[string]string
}
type Index struct {
	mu     sync.RWMutex
	chunks map[string]Chunk
}

func NewIndex() *Index { return &Index{chunks: map[string]Chunk{}} }
func (i *Index) Ingest(_ context.Context, d Document, chunkSize int) error {
	if d.ID == "" || d.TenantID == "" || d.Text == "" {
		return errors.New("document id, tenant and text required")
	}
	if chunkSize <= 0 {
		chunkSize = 1200
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	for n, start := 0, 0; start < len(d.Text); n++ {
		end := start + chunkSize
		if end > len(d.Text) {
			end = len(d.Text)
		}
		id := d.ID + ":" + string(rune(n))
		i.chunks[id] = Chunk{ID: id, DocumentID: d.ID, Text: d.Text[start:end], TenantID: d.TenantID, ProjectID: d.ProjectID, Metadata: d.Metadata}
		start = end
	}
	return nil
}

type Hit struct {
	Chunk Chunk
	Score float64
}

func (i *Index) Search(_ context.Context, tenant, project, q string, limit int) []Hit {
	if limit <= 0 {
		limit = 8
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	var hs []Hit
	for _, c := range i.chunks {
		if c.TenantID != tenant || c.ProjectID != project {
			continue
		}
		score := float64(overlap(q, c.Text))
		if score > 0 {
			hs = append(hs, Hit{c, score})
		}
	}
	sort.Slice(hs, func(a, b int) bool { return hs[a].Score > hs[b].Score })
	if len(hs) > limit {
		hs = hs[:limit]
	}
	return hs
}
func overlap(a, b string) int {
	as := map[string]bool{}
	for _, w := range words(a) {
		as[w] = true
	}
	n := 0
	for _, w := range words(b) {
		if as[w] {
			n++
		}
	}
	return n
}
func words(s string) []string {
	var o []string
	w := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if r >= 'A' && r <= 'Z' {
				r += 32
			}
			w += string(r)
		} else if w != "" {
			o = append(o, w)
			w = ""
		}
	}
	if w != "" {
		o = append(o, w)
	}
	return o
}
