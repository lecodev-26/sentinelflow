package configv4

import (
	"os"
	"strconv"
	"strings"
)

type Flags struct{ RoutingV4, FinOps, EventBus, Enterprise, SecurityCenter bool }

func LoadFlags() Flags {
	return Flags{RoutingV4: enabled("SF_FEATURE_ROUTING_V4", true), FinOps: enabled("SF_FEATURE_FINOPS", true), EventBus: enabled("SF_FEATURE_EVENT_BUS", true), Enterprise: enabled("SF_FEATURE_ENTERPRISE", true), SecurityCenter: enabled("SF_FEATURE_SECURITY_CENTER", true)}
}
func enabled(name string, def bool) bool {
	v, ok := os.LookupEnv(name)
	if !ok {
		return def
	}
	b, e := strconv.ParseBool(strings.TrimSpace(v))
	if e != nil {
		return def
	}
	return b
}
