package regions

import "testing"

func TestDecideStorage(t *testing.T) {
	r := NewResolver()
	if d := r.DecideStorage(ResidencyEU, EUWest); !d.Allowed {
		t.Fatal(d.Reason)
	}
	if d := r.DecideStorage(ResidencyEU, USEast); d.Allowed {
		t.Fatal("US must not store EU-resident data")
	}
	if d := r.DecideStorage(ResidencyUS, USEast); !d.Allowed {
		t.Fatal(d.Reason)
	}
}
