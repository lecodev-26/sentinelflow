package globalv5

import (
	"sort"
	"time"
)

type Region struct {
	ID        string
	Latency   time.Duration
	Capacity  float64
	Healthy   bool
	Residency []string
}
type Router struct{ regions map[string]Region }

func NewRouter(rs []Region) *Router {
	m := map[string]Region{}
	for _, r := range rs {
		m[r.ID] = r
	}
	return &Router{regions: m}
}
func (r *Router) Select(allowed []string) []Region {
	var out []Region
	for _, x := range r.regions {
		if !x.Healthy || x.Capacity <= 0 {
			continue
		}
		if len(allowed) > 0 && !contains(allowed, x.ID) {
			continue
		}
		out = append(out, x)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Latency == out[j].Latency {
			return out[i].Capacity > out[j].Capacity
		}
		return out[i].Latency < out[j].Latency
	})
	return out
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
