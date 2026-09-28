package regions

import "fmt"

type ResidencyDecision struct {
	Allowed   bool           `json:"allowed"`
	Residency ResidencyLevel `json:"residency"`
	Region    Region         `json:"region"`
	Reason    string         `json:"reason"`
}

func (r *Resolver) DecideStorage(residency ResidencyLevel, region Region) ResidencyDecision {
	if residency == "" {
		residency = ResidencyGlobal
	}
	if err := r.ValidateRegion(residency, region); err != nil {
		return ResidencyDecision{Allowed: false, Residency: residency, Region: region, Reason: err.Error()}
	}
	p, _ := r.GetPolicy(residency)
	if p != nil && p.RequireLocalStorage {
		return ResidencyDecision{Allowed: true, Residency: residency, Region: region, Reason: "local_storage_required_and_region_allowed"}
	}
	return ResidencyDecision{Allowed: true, Residency: residency, Region: region, Reason: fmt.Sprintf("region_allowed_for_%s", residency)}
}
