package v3

import (
	regions "github.com/lecodev-26/sentinelflow/internal/enterprise/v3/regions"
	"testing"
)

func TestMultiRegionResidency(t *testing.T) {
	r := NewMultiRegionRouter(regions.NewResolver())
	r.SetProviderRegions("openai", []regions.Region{regions.EUWest, regions.USEast})
	got := r.Eligible("openai", regions.ResidencyEU)
	if len(got) != 1 || got[0] != regions.EUWest {
		t.Fatalf("unexpected EU placement: %#v", got)
	}
}
