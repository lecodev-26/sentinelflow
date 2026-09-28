package approvals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lecodev-26/sentinelflow/internal/events"
	"github.com/lecodev-26/sentinelflow/internal/idgen"
	"strings"
	"time"
)

const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusExpired   = "expired"
	StatusCancelled = "cancelled"
)

type Request struct {
	ID, TenantID, ProjectID, Environment, RequesterID           string
	Action, TargetType, TargetID, PayloadSHA256, Reason, Status string
	ExpiresAt                                                   *time.Time
	ApproverID, DecisionReason                                  string
	CreatedAt                                                   time.Time
	DecidedAt, UpdatedAt                                        *time.Time
}

func (r Request) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"id": r.ID, "tenant_id": r.TenantID, "project_id": r.ProjectID,
		"environment": r.Environment, "requester_id": r.RequesterID, "action": r.Action,
		"target_type": r.TargetType, "target_id": r.TargetID, "payload_sha256": r.PayloadSHA256,
		"reason": r.Reason, "status": r.Status, "expires_at": r.ExpiresAt,
		"approver_id": r.ApproverID, "decision_reason": r.DecisionReason,
		"created_at": r.CreatedAt, "decided_at": r.DecidedAt, "updated_at": r.UpdatedAt,
	})
}

type CreateInput struct {
	TenantID, ProjectID, Environment, RequesterID string
	Action, TargetType, TargetID, Reason          string
	Payload                                       any
	ExpiresIn                                     time.Duration
}
type Service struct {
	pool   *pgxpool.Pool
	outbox *events.Outbox
}

func NewService(pool *pgxpool.Pool, outbox *events.Outbox) *Service {
	return &Service{pool: pool, outbox: outbox}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*Request, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.Action) == "" {
		return nil, errors.New("tenant_id and action are required")
	}
	if in.Environment == "" {
		in.Environment = "production"
	}
	if !validEnv(in.Environment) {
		return nil, fmt.Errorf("invalid environment: %s", in.Environment)
	}
	if in.ExpiresIn <= 0 {
		in.ExpiresIn = 30 * time.Minute
	}
	if in.ExpiresIn > 7*24*time.Hour {
		return nil, errors.New("expires_in cannot exceed 7 days")
	}
	b, e := json.Marshal(in.Payload)
	if e != nil {
		return nil, e
	}
	sum := sha256.Sum256(b)
	hash := hex.EncodeToString(sum[:])
	now := time.Now().UTC()
	exp := now.Add(in.ExpiresIn)
	req := &Request{ID: "apr_" + idgen.RandomHex(12), TenantID: in.TenantID, ProjectID: in.ProjectID, Environment: in.Environment, RequesterID: in.RequesterID, Action: in.Action, TargetType: in.TargetType, TargetID: in.TargetID, PayloadSHA256: hash, Reason: in.Reason, Status: StatusPending, ExpiresAt: &exp, CreatedAt: now, UpdatedAt: &now}
	e = withTx(ctx, s.pool, func(t pgx.Tx) error {
		_, e := t.Exec(ctx, "INSERT INTO approval_requests (id,tenant_id,project_id,environment,requester_id,action,target_type,target_id,payload_sha256,reason,status,expires_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13)", req.ID, req.TenantID, req.ProjectID, req.Environment, req.RequesterID, req.Action, req.TargetType, req.TargetID, req.PayloadSHA256, req.Reason, req.Status, req.ExpiresAt, req.CreatedAt)
		if e != nil {
			return e
		}
		if s.outbox != nil {
			ev := events.NewEvent(events.EventApprovalRequested).WithTenant(req.TenantID).WithProject(req.ProjectID).WithUser(req.RequesterID).WithPayload("approval_id", req.ID).WithPayload("environment", req.Environment).WithPayload("action", req.Action).WithPayload("target_type", req.TargetType).WithPayload("target_id", req.TargetID).WithPayload("payload_sha256", req.PayloadSHA256)
			return s.outbox.EnqueueTx(ctx, t, ev)
		}
		return nil
	})
	if e != nil {
		return nil, fmt.Errorf("create approval: %w", e)
	}
	return req, nil
}

const selectApproval = "SELECT id,tenant_id,project_id,environment,requester_id,action,target_type,target_id,payload_sha256,reason,status,expires_at,approver_id,decision_reason,created_at,decided_at,updated_at FROM approval_requests "

type nullableTime struct{ dst **time.Time }

func (n nullableTime) Scan(src any) error {
	if src == nil {
		*n.dst = nil
		return nil
	}
	t, ok := src.(time.Time)
	if !ok {
		return fmt.Errorf("expected time.Time, got %T", src)
	}
	*n.dst = &t
	return nil
}

type nullableString struct{ dst *string }

func (n nullableString) Scan(src any) error {
	if src == nil {
		*n.dst = ""
		return nil
	}
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", src)
	}
	*n.dst = s
	return nil
}

func scan(r *Request) []any {
	return []any{&r.ID, &r.TenantID, &r.ProjectID, &r.Environment, &r.RequesterID, &r.Action, &r.TargetType, &r.TargetID, &r.PayloadSHA256, &r.Reason, &r.Status, nullableTime{&r.ExpiresAt}, nullableString{&r.ApproverID}, &r.DecisionReason, &r.CreatedAt, nullableTime{&r.DecidedAt}, nullableTime{&r.UpdatedAt}}
}

func (s *Service) Get(ctx context.Context, tenant, env, id string) (*Request, error) {
	var r Request
	e := s.pool.QueryRow(ctx, selectApproval+"WHERE id=$1 AND tenant_id=$2 AND environment=$3", id, tenant, env).Scan(scan(&r)...)
	if e != nil {
		return nil, e
	}
	return &r, nil
}

func (s *Service) List(ctx context.Context, tenant, env, status string, limit int) ([]Request, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args := []any{tenant, env}
	q := selectApproval + "WHERE tenant_id=$1 AND environment=$2"
	if status != "" {
		if !validStatus(status) {
			return nil, errors.New("invalid status")
		}
		args = append(args, status)
		q += " AND status=$3"
	}
	q += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", limit)
	rows, e := s.pool.Query(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Request{}
	for rows.Next() {
		var r Request
		if e := rows.Scan(scan(&r)...); e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) Decide(ctx context.Context, tenant, env, id, approver, status, reason string) (*Request, error) {
	if status != StatusApproved && status != StatusRejected {
		return nil, errors.New("decision must be approved or rejected")
	}
	if approver == "" {
		return nil, errors.New("approver_id is required")
	}
	var req Request
	e := withTx(ctx, s.pool, func(t pgx.Tx) error {
		if e := t.QueryRow(ctx, selectApproval+"WHERE id=$1 AND tenant_id=$2 AND environment=$3 FOR UPDATE", id, tenant, env).Scan(scan(&req)...); e != nil {
			return e
		}
		if req.Status != StatusPending {
			return fmt.Errorf("approval is already %s", req.Status)
		}
		if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now().UTC()) {
			return expire(t, ctx, &req)
		}
		if req.RequesterID != "" && req.RequesterID == approver {
			return errors.New("requester cannot approve or reject their own request")
		}
		now := time.Now().UTC()
		if _, e := t.Exec(ctx, "UPDATE approval_requests SET status=$1,approver_id=$2,decision_reason=$3,decided_at=$4,updated_at=$4 WHERE id=$5", status, approver, reason, now, id); e != nil {
			return e
		}
		req.Status = status
		req.ApproverID = approver
		req.DecisionReason = reason
		req.DecidedAt = &now
		req.UpdatedAt = &now
		if s.outbox != nil {
			typ := events.EventApprovalApproved
			if status == StatusRejected {
				typ = events.EventApprovalRejected
			}
			ev := events.NewEvent(typ).WithTenant(req.TenantID).WithProject(req.ProjectID).WithUser(approver).WithPayload("approval_id", req.ID).WithPayload("environment", req.Environment).WithPayload("action", req.Action).WithPayload("payload_sha256", req.PayloadSHA256).WithPayload("decision_reason", reason).WithPayload("requester_id", req.RequesterID)
			return s.outbox.EnqueueTx(ctx, t, ev)
		}
		return nil
	})
	if e != nil {
		return nil, fmt.Errorf("decide approval: %w", e)
	}
	if req.Status == StatusExpired {
		return nil, errors.New("approval expired")
	}
	return &req, nil
}

func (s *Service) Cancel(ctx context.Context, tenant, env, id, actor string) error {
	return withTx(ctx, s.pool, func(t pgx.Tx) error {
		var requester, current, project, action, hash string
		if e := t.QueryRow(ctx, "SELECT requester_id,status,project_id,action,payload_sha256 FROM approval_requests WHERE id=$1 AND tenant_id=$2 AND environment=$3 FOR UPDATE", id, tenant, env).Scan(&requester, &current, &project, &action, &hash); e != nil {
			return e
		}
		if current != StatusPending {
			return fmt.Errorf("approval is already %s", current)
		}
		if requester != actor {
			return errors.New("only the requester can cancel a pending approval")
		}
		if _, e := t.Exec(ctx, "UPDATE approval_requests SET status='cancelled',updated_at=NOW(),decided_at=NOW() WHERE id=$1", id); e != nil {
			return e
		}
		if s.outbox != nil {
			ev := events.NewEvent(events.EventApprovalCancelled).WithTenant(tenant).WithProject(project).WithUser(actor).WithPayload("approval_id", id).WithPayload("environment", env).WithPayload("action", action).WithPayload("payload_sha256", hash)
			return s.outbox.EnqueueTx(ctx, t, ev)
		}
		return nil
	})
}

func expire(t pgx.Tx, ctx context.Context, r *Request) error {
	now := time.Now().UTC()
	if _, e := t.Exec(ctx, "UPDATE approval_requests SET status='expired',decided_at=$1,updated_at=$1 WHERE id=$2", now, r.ID); e != nil {
		return e
	}
	r.Status = StatusExpired
	r.DecidedAt = &now
	r.UpdatedAt = &now
	return nil
}
func withTx(ctx context.Context, p *pgxpool.Pool, fn func(pgx.Tx) error) error {
	t, e := p.Begin(ctx)
	if e != nil {
		return e
	}
	defer t.Rollback(ctx)
	if e = fn(t); e != nil {
		return e
	}
	return t.Commit(ctx)
}
func validEnv(v string) bool { return v == "development" || v == "staging" || v == "production" }
func validStatus(v string) bool {
	switch v {
	case StatusPending, StatusApproved, StatusRejected, StatusExpired, StatusCancelled:
		return true
	}
	return false
}
