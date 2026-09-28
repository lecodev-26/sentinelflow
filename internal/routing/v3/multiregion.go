package v3

import (
	regions "github.com/lecodev-26/sentinelflow/internal/enterprise/v3/regions"
	"sort"
)

type RegionRoute struct {
	Provider string         `json:"provider"`
	Region   regions.Region `json:"region"`
	Healthy  bool           `json:"healthy"`
}

type MultiRegionRouter struct {
	placements map[string][]regions.Region
	resolver   *regions.Resolver
}

func NewMultiRegionRouter(resolver *regions.Resolver) *MultiRegionRouter {
	return &MultiRegionRouter{placements: map[string][]regions.Region{}, resolver: resolver}
}

func (r *MultiRegionRouter) SetProviderRegions(provider string, rs []regions.Region) {
	r.placements[provider] = append([]regions.Region(nil), rs...)
}

func (r *MultiRegionRouter) Regions(provider string) []regions.Region {
	return append([]regions.Region(nil), r.placements[provider]...)
}

func (r *MultiRegionRouter) Eligible(provider string, residency regions.ResidencyLevel) []regions.Region {
	out := []regions.Region{}
	for _, reg := range r.placements[provider] {
		if r.resolver.ValidateRegion(residency, reg) == nil {
			out = append(out, reg)
		}
	}
	return out
}

func (r *MultiRegionRouter) Rank(routes []RegionRoute) []RegionRoute {
	out := append([]RegionRoute(nil), routes...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Healthy != out[j].Healthy {
			return out[i].Healthy
		}
		return string(out[i].Region) < string(out[j].Region)
	})
	return out
}
