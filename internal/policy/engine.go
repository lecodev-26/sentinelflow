package policy

import (
"context"
"fmt"
"sync"
"time"
)

type Policy struct {
ID          string                 `json:"id"`
Name        string                 `json:"name"`
Description string                 `json:"description"`
Priority    int                    `json:"priority"`
Enabled     bool                   `json:"enabled"`
Conditions  []Condition            `json:"conditions"`
Actions     []Action               `json:"actions"`
Metadata    map[string]interface{} `json:"metadata"`
}

type Condition struct {
Type   string      `json:"type"`
Field  string      `json:"field"`
Op     string      `json:"op"`
Value  interface{} `json:"value"`
}

type Action struct {
Type   string                 `json:"type"`
Params map[string]interface{} `json:"params"`
}

type EvaluationContext struct {
RequestID  string                 `json:"request_id"`
TenantID   string                 `json:"tenant_id"`
UserID     string                 `json:"user_id"`
Provider   string                 `json:"provider"`
Model      string                 `json:"model"`
Method     string                 `json:"method"`
Path       string                 `json:"path"`
Input      interface{}            `json:"input"`
Output     interface{}            `json:"output"`
Metadata   map[string]interface{} `json:"metadata"`
}

type EvaluationResult struct {
Allowed    bool                   `json:"allowed"`
Actions    []Action               `json:"actions"`
PolicyIDs  []string               `json:"policy_ids"`
Reason     string                 `json:"reason"`
Metadata   map[string]interface{} `json:"metadata"`
}

type Engine struct {
mu       sync.RWMutex
policies map[string]Policy
order    []string
}

func NewEngine() *Engine {
return &Engine{
policies: make(map[string]Policy),
order:    []string{},
}
}

func (e *Engine) AddPolicy(policy Policy) {
e.mu.Lock()
defer e.mu.Unlock()

if policy.Priority == 0 {
policy.Priority = 100
}
if policy.ID == "" {
policy.ID = generateID()
}

e.policies[policy.ID] = policy
e.order = append(e.order, policy.ID)

for i := 0; i < len(e.order); i++ {
for j := i + 1; j < len(e.order); j++ {
if e.policies[e.order[i]].Priority > e.policies[e.order[j]].Priority {
e.order[i], e.order[j] = e.order[j], e.order[i]
}
}
}
}

func (e *Engine) RemovePolicy(id string) bool {
e.mu.Lock()
defer e.mu.Unlock()

if _, exists := e.policies[id]; !exists {
return false
}

delete(e.policies, id)

newOrder := []string{}
for _, pid := range e.order {
if pid != id {
newOrder = append(newOrder, pid)
}
}
e.order = newOrder
return true
}

func (e *Engine) GetPolicy(id string) (Policy, bool) {
e.mu.RLock()
defer e.mu.RUnlock()

p, exists := e.policies[id]
return p, exists
}

func (e *Engine) ListPolicies() []Policy {
e.mu.RLock()
defer e.mu.RUnlock()

result := make([]Policy, 0, len(e.order))
for _, id := range e.order {
if p, exists := e.policies[id]; exists {
result = append(result, p)
}
}
return result
}

func (e *Engine) Evaluate(ctx context.Context, evalCtx *EvaluationContext) EvaluationResult {
e.mu.RLock()
defer e.mu.RUnlock()

result := EvaluationResult{
Allowed:   true,
Actions:   []Action{},
PolicyIDs: []string{},
Metadata:  make(map[string]interface{}),
}

for _, id := range e.order {
policy, exists := e.policies[id]
if !exists || !policy.Enabled {
continue
}

if e.evaluatePolicy(evalCtx, policy) {
result.PolicyIDs = append(result.PolicyIDs, policy.ID)

for _, action := range policy.Actions {
result.Actions = append(result.Actions, action)
if action.Type == "deny" {
result.Allowed = false
result.Reason = fmt.Sprintf("Policy %s: %s", policy.Name, policy.Description)
}
}
}
}

return result
}

func (e *Engine) evaluatePolicy(ctx *EvaluationContext, policy Policy) bool {
for _, condition := range policy.Conditions {
if !e.evaluateCondition(ctx, condition) {
return false
}
}
return true
}

func (e *Engine) evaluateCondition(ctx *EvaluationContext, cond Condition) bool {
value := e.getValue(ctx, cond.Field)
return compare(value, cond.Op, cond.Value)
}

func (e *Engine) getValue(ctx *EvaluationContext, field string) interface{} {
switch field {
case "tenant_id":
return ctx.TenantID
case "user_id":
return ctx.UserID
case "provider":
return ctx.Provider
case "model":
return ctx.Model
case "method":
return ctx.Method
case "path":
return ctx.Path
default:
if ctx.Metadata != nil {
if val, ok := ctx.Metadata[field]; ok {
return val
}
}
return nil
}
}

func compare(a interface{}, op string, b interface{}) bool {
switch op {
case "eq":
return a == b
case "neq":
return a != b
case "gt":
return compareNumeric(a, b) > 0
case "gte":
return compareNumeric(a, b) >= 0
case "lt":
return compareNumeric(a, b) < 0
case "lte":
return compareNumeric(a, b) <= 0
case "in":
return isIn(a, b)
case "contains":
return contains(a, b)
case "starts_with":
return startsWith(a, b)
case "ends_with":
return endsWith(a, b)
default:
return false
}
}

func compareNumeric(a, b interface{}) float64 {
af, aok := toFloat(a)
bf, bok := toFloat(b)
if !aok || !bok {
return 0
}
return af - bf
}

func toFloat(v interface{}) (float64, bool) {
switch val := v.(type) {
case int:
return float64(val), true
case int64:
return float64(val), true
case float64:
return val, true
case float32:
return float64(val), true
default:
return 0, false
}
}

func isIn(a, b interface{}) bool {
switch v := b.(type) {
case []string:
s, ok := a.(string)
if !ok {
return false
}
for _, item := range v {
if item == s {
return true
}
}
case []interface{}:
for _, item := range v {
if item == a {
return true
}
}
}
return false
}

func contains(a, b interface{}) bool {
s, ok := a.(string)
if !ok {
return false
}
sub, ok := b.(string)
if !ok {
return false
}
return len(s) >= len(sub) && (s == sub || len(s) > len(sub) && (s[:len(sub)] == sub || s[len(s)-len(sub):] == sub || containsMiddle(s, sub)))
}

func containsMiddle(s, sub string) bool {
for i := 1; i <= len(s)-len(sub)-1; i++ {
if s[i:i+len(sub)] == sub {
return true
}
}
return false
}

func startsWith(a, b interface{}) bool {
s, ok := a.(string)
if !ok {
return false
}
prefix, ok := b.(string)
if !ok {
return false
}
return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func endsWith(a, b interface{}) bool {
s, ok := a.(string)
if !ok {
return false
}
suffix, ok := b.(string)
if !ok {
return false
}
return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func generateID() string {
return "pol_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

func randomString(n int) string {
const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
b := make([]byte, n)
for i := range b {
b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
}
return string(b)
}
