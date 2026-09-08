package audit

import (
"bytes"
"encoding/json"
"os"
"sync"
"time"
)

// Level representa el nivel de auditoría
type Level string

const (
LevelInfo  Level = "info"
LevelWarn  Level = "warn"
LevelError Level = "error"
LevelBlock Level = "block"
)

// Entry representa una entrada de auditoría
type Entry struct {
ID        string                 `json:"id"`
Timestamp time.Time              `json:"timestamp"`
Level     Level                  `json:"level"`
TenantID  string                 `json:"tenant_id"`
UserID    string                 `json:"user_id,omitempty"`
RequestID string                 `json:"request_id"`
Method    string                 `json:"method"`
Path      string                 `json:"path"`
Status    int                    `json:"status"`
Provider  string                 `json:"provider,omitempty"`
Model     string                 `json:"model,omitempty"`
Tokens    int                    `json:"tokens,omitempty"`
Cost      float64                `json:"cost,omitempty"`
Latency   time.Duration          `json:"latency_ms"`
IP        string                 `json:"ip,omitempty"`
UserAgent string                 `json:"user_agent,omitempty"`
Error     string                 `json:"error,omitempty"`
Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Logger es el sistema de auditoría
type Logger struct {
mu          sync.Mutex
file        *os.File
entries     []Entry
maxEntries  int
retention   time.Duration
outputFile  string
}

// Config configuración del logger
type Config struct {
MaxEntries int
Retention  time.Duration
OutputFile string
}

// DefaultConfig devuelve la configuración por defecto
func DefaultConfig() Config {
return Config{
MaxEntries: 10000,
Retention:  30 * 24 * time.Hour,
OutputFile: "logs/audit.jsonl",
}
}

// NewLogger crea un nuevo logger de auditoría
func NewLogger(config Config) (*Logger, error) {
logger := &Logger{
entries:    make([]Entry, 0),
maxEntries: config.MaxEntries,
retention:  config.Retention,
outputFile: config.OutputFile,
}

if err := os.MkdirAll("logs", 0755); err != nil {
return nil, err
}

file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
if err != nil {
return nil, err
}
logger.file = file

if err := logger.load(); err != nil {
// No fallar si no hay logs
}

return logger, nil
}

func (l *Logger) load() error {
l.mu.Lock()
defer l.mu.Unlock()

data, err := os.ReadFile(l.outputFile)
if err != nil {
if os.IsNotExist(err) {
return nil
}
return err
}

var entries []Entry
for _, line := range bytes.Split(data, []byte("\n")) {
if len(line) == 0 {
continue
}
var entry Entry
if err := json.Unmarshal(line, &entry); err == nil {
entries = append(entries, entry)
}
}
l.entries = entries
return nil
}

// Log registra una entrada de auditoría
func (l *Logger) Log(entry Entry) error {
l.mu.Lock()
defer l.mu.Unlock()

entry.Timestamp = time.Now()
if entry.ID == "" {
entry.ID = generateID()
}

l.entries = append(l.entries, entry)

if len(l.entries) > l.maxEntries {
l.entries = l.entries[len(l.entries)-l.maxEntries:]
}

if l.retention > 0 {
cutoff := time.Now().Add(-l.retention)
var filtered []Entry
for _, e := range l.entries {
if e.Timestamp.After(cutoff) {
filtered = append(filtered, e)
}
}
l.entries = filtered
}

data, err := json.Marshal(entry)
if err != nil {
return err
}
_, err = l.file.Write(append(data, '\n'))
return err
}

// LogRequest registra una petición
func (l *Logger) LogRequest(tenantID, requestID, method, path string, status int, latency time.Duration) error {
return l.Log(Entry{
TenantID:  tenantID,
RequestID: requestID,
Method:    method,
Path:      path,
Status:    status,
Latency:   latency,
Level:     LevelInfo,
})
}

// LogBlock registra un bloqueo
func (l *Logger) LogBlock(tenantID, requestID, reason, provider string) error {
return l.Log(Entry{
TenantID:  tenantID,
RequestID: requestID,
Status:    403,
Level:     LevelBlock,
Error:     reason,
Provider:  provider,
})
}

// Query busca entradas de auditoría
func (l *Logger) Query(filter Filter) []Entry {
l.mu.Lock()
defer l.mu.Unlock()

var result []Entry
for _, entry := range l.entries {
if filter.Match(entry) {
result = append(result, entry)
}
}
return result
}

// Close cierra el logger
func (l *Logger) Close() error {
if l.file != nil {
return l.file.Close()
}
return nil
}

// Filter para consultas de auditoría
type Filter struct {
TenantID  string
UserID    string
RequestID string
Method    string
Path      string
Status    int
Level     Level
Provider  string
Model     string
From      time.Time
To        time.Time
Limit     int
}

// Match verifica si una entrada coincide con el filtro
func (f *Filter) Match(entry Entry) bool {
if f.TenantID != "" && entry.TenantID != f.TenantID {
return false
}
if f.UserID != "" && entry.UserID != f.UserID {
return false
}
if f.RequestID != "" && entry.RequestID != f.RequestID {
return false
}
if f.Method != "" && entry.Method != f.Method {
return false
}
if f.Path != "" && entry.Path != f.Path {
return false
}
if f.Status != 0 && entry.Status != f.Status {
return false
}
if f.Level != "" && entry.Level != f.Level {
return false
}
if f.Provider != "" && entry.Provider != f.Provider {
return false
}
if f.Model != "" && entry.Model != f.Model {
return false
}
if !f.From.IsZero() && entry.Timestamp.Before(f.From) {
return false
}
if !f.To.IsZero() && entry.Timestamp.After(f.To) {
return false
}
return true
}

func generateID() string {
return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
b := make([]byte, n)
for i := range b {
b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
}
return string(b)
}
