package v3

import (
	"sync"
	"time"
)

type Anomaly struct {
	ID        string        `json:"id"`
	TenantID  string        `json:"tenant_id"`
	APIKeyID  string        `json:"api_key_id,omitempty"`
	Kind      string        `json:"kind"`
	Severity  RiskLevel     `json:"severity"`
	Count     int           `json:"count"`
	Window    time.Duration `json:"window"`
	Reason    string        `json:"reason"`
	FirstSeen time.Time     `json:"first_seen"`
	LastSeen  time.Time     `json:"last_seen"`
}

type AnomalyDetector struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	anomalies []Anomaly
	max       int
	window    time.Duration
	burst     int
}

type bucket struct {
	first, last           time.Time
	total, errors, blocks int
}

func NewAnomalyDetector(window time.Duration, burst, max int) *AnomalyDetector {
	if window <= 0 {
		window = 1 * time.Minute
	}
	if burst <= 0 {
		burst = 60
	}
	if max <= 0 {
		max = 1000
	}
	return &AnomalyDetector{buckets: map[string]*bucket{}, max: max, window: window, burst: burst}
}

func (d *AnomalyDetector) Record(tenant, key string, status int, blocked bool, now time.Time) *Anomaly {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := tenant + ":" + key
	b := d.buckets[id]
	if b == nil || now.Sub(b.first) > d.window {
		b = &bucket{first: now}
		d.buckets[id] = b
	}
	b.last = now
	b.total++
	if status >= 400 {
		b.errors++
	}
	if blocked {
		b.blocks++
	}
	if b.total < d.burst && b.errors < d.burst/2 && b.blocks < d.burst/4 {
		return nil
	}
	kind, sev, reason := "burst", RiskMedium, "request volume exceeded anomaly threshold"
	if b.blocks >= d.burst/4 {
		kind, sev, reason = "security_blocks", RiskHigh, "repeated security policy blocks"
	}
	if b.errors >= d.burst/2 {
		kind, sev, reason = "error_burst", RiskHigh, "repeated gateway errors"
	}
	a := Anomaly{ID: "anom-" + now.Format("20060102150405.000000000"), TenantID: tenant, APIKeyID: key, Kind: kind, Severity: sev, Count: b.total, Window: d.window, Reason: reason, FirstSeen: b.first, LastSeen: b.last}
	d.anomalies = append(d.anomalies, a)
	if len(d.anomalies) > d.max {
		d.anomalies = d.anomalies[len(d.anomalies)-d.max:]
	}
	return &a
}

func (d *AnomalyDetector) List(tenant string, limit int) []Anomaly {
	d.mu.Lock()
	defer d.mu.Unlock()
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	out := make([]Anomaly, 0, limit)
	for i := len(d.anomalies) - 1; i >= 0 && len(out) < limit; i-- {
		if tenant == "" || d.anomalies[i].TenantID == tenant {
			out = append(out, d.anomalies[i])
		}
	}
	return out
}
