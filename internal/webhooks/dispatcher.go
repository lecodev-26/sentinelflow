package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/lecodev-26/sentinelflow/internal/metrics"
)

// Dispatcher lee webhook_deliveries pendientes y las envía.
type Dispatcher struct {
	pool         *pgxpool.Pool
	httpClient   *http.Client
	pollInterval time.Duration
	batchSize    int
	maxAttempts  int
	baseBackoff  time.Duration
	maxBackoff   time.Duration
}

// DispatcherConfig configura el dispatcher.
type DispatcherConfig struct {
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  int
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
	HTTPTimeout  time.Duration
}

func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		PollInterval: 2 * time.Second,
		BatchSize:    50,
		MaxAttempts:  8,
		BaseBackoff:  5 * time.Second,
		MaxBackoff:   10 * time.Minute,
		HTTPTimeout:  10 * time.Second,
	}
}

func NewDispatcher(pool *pgxpool.Pool, cfg DispatcherConfig) *Dispatcher {
	return &Dispatcher{
		pool:         pool,
		httpClient:   &http.Client{Timeout: cfg.HTTPTimeout},
		pollInterval: cfg.PollInterval,
		batchSize:    cfg.BatchSize,
		maxAttempts:  cfg.MaxAttempts,
		baseBackoff:  cfg.BaseBackoff,
		maxBackoff:   cfg.MaxBackoff,
	}
}

// Run bloquea hasta que el ctx se cancele.
func (d *Dispatcher) Run(ctx context.Context) error {
	logrus.WithFields(logrus.Fields{
		"poll_interval": d.pollInterval,
		"batch_size":    d.batchSize,
	}).Info("🔗 WebhookDispatcher iniciado")

	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logrus.Info("🔗 WebhookDispatcher detenido")
			return ctx.Err()
		case <-ticker.C:
			if err := d.drain(ctx); err != nil {
				logrus.Errorf("webhooks: drain error: %v", err)
			}
		}
	}
}

func (d *Dispatcher) drain(ctx context.Context) error {
	for {
		n, err := d.processBatch(ctx)
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
	}
}

type deliveryRow struct {
	id        string
	webhookID string
	url       string
	secret    string
	eventType string
	payload   []byte
	attempts  int
}

func (d *Dispatcher) processBatch(ctx context.Context) (int, error) {
	tx, err := d.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := d.claimBatch(ctx, tx)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, tx.Commit(ctx)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	for _, r := range rows {
		d.deliverOne(ctx, r)
	}
	return len(rows), nil
}

func (d *Dispatcher) claimBatch(ctx context.Context, tx pgx.Tx) ([]deliveryRow, error) {
	const q = `
WITH claimed AS (
SELECT d.id
FROM webhook_deliveries d
WHERE d.status IN ('pending','failed')
  AND d.next_attempt_at <= NOW()
ORDER BY d.next_attempt_at
LIMIT $1
FOR UPDATE SKIP LOCKED
)
UPDATE webhook_deliveries d
SET status   = 'delivering',
    attempts = d.attempts + 1,
    updated_at = NOW()
FROM claimed, webhooks w
WHERE d.id = claimed.id
  AND w.id = d.webhook_id
RETURNING
d.id, d.webhook_id, w.url, w.secret,
d.event_type, d.payload, d.attempts
`
	rows, err := tx.Query(ctx, q, d.batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []deliveryRow
	for rows.Next() {
		var r deliveryRow
		if err := rows.Scan(
			&r.id, &r.webhookID, &r.url, &r.secret,
			&r.eventType, &r.payload, &r.attempts,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (d *Dispatcher) deliverOne(ctx context.Context, r deliveryRow) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(r.payload))
	if err != nil {
		d.markFailed(ctx, r.id, r.attempts, 0, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SF-Event-Type", r.eventType)
	req.Header.Set("X-SF-Delivery", r.id)
	req.Header.Set("X-SF-Signature", sign(r.payload, r.secret))

	resp, err := d.httpClient.Do(req)
	if err != nil {
		d.markFailed(ctx, r.id, r.attempts, 0, err.Error())
		metrics.HTTPRequestsTotal.WithLabelValues("POST", "webhook", "error").Inc()
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		d.markDelivered(ctx, r.id, resp.StatusCode)
		metrics.HTTPRequestsTotal.WithLabelValues("POST", "webhook", "ok").Inc()
		return
	}
	d.markFailed(ctx, r.id, r.attempts, resp.StatusCode,
		fmt.Sprintf("http %d", resp.StatusCode))
	metrics.HTTPRequestsTotal.WithLabelValues("POST", "webhook", "failed").Inc()
}

func (d *Dispatcher) markDelivered(ctx context.Context, id string, code int) {
	const q = `
UPDATE webhook_deliveries
SET status='delivered', response_code=$2, delivered_at=NOW(), updated_at=NOW()
WHERE id=$1
`
	if _, err := d.pool.Exec(ctx, q, id, code); err != nil {
		logrus.Errorf("webhooks: markDelivered %s: %v", id, err)
	}
}

func (d *Dispatcher) markFailed(ctx context.Context, id string, attempts, code int, msg string) {
	status := "failed"
	if attempts >= d.maxAttempts {
		status = "dead"
	}
	backoff := d.computeBackoff(attempts)
	next := time.Now().Add(backoff)

	const q = `
UPDATE webhook_deliveries
SET status=$2, response_code=$3, last_error=$4,
    next_attempt_at=$5, updated_at=NOW()
WHERE id=$1
`
	if _, err := d.pool.Exec(ctx, q, id, status, code, msg, next); err != nil {
		logrus.Errorf("webhooks: markFailed %s: %v", id, err)
		return
	}
	if status == "dead" {
		logrus.Errorf("☠️ webhooks: delivery %s dead tras %d intentos: %s", id, attempts, msg)
	}
}

func (d *Dispatcher) computeBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		attempts = 1
	}
	backoff := d.baseBackoff
	for i := 1; i < attempts; i++ {
		backoff *= 2
		if backoff >= d.maxBackoff {
			backoff = d.maxBackoff
			break
		}
	}
	// jitter ±20%
	delta := float64(backoff) * 0.2
	offset := (rand.Float64()*2 - 1) * delta
	backoff = time.Duration(float64(backoff) + offset)
	if backoff < 0 {
		backoff = 0
	}
	return backoff
}

// sign calcula HMAC-SHA256 del payload con el secret del webhook.
// El cliente valida con el mismo secret.
func sign(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// helper para asegurar uso de encoding/json (por si el refactor elimina referencias)
var _ = json.Marshal
