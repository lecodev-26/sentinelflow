package postgres

import (
"context"
"fmt"
"time"

"github.com/jackc/pgx/v5"
"github.com/jackc/pgx/v5/pgxpool"
)

// Config contiene la configuración del cliente PostgreSQL
type Config struct {
URL             string
MaxConns        int32
MinConns        int32
MaxConnLifetime time.Duration
MaxConnIdleTime time.Duration
ConnectTimeout  time.Duration
}

// DefaultConfig devuelve la configuración por defecto
func DefaultConfig(url string) Config {
return Config{
URL:             url,
MaxConns:        25,
MinConns:        5,
MaxConnLifetime: 30 * time.Minute,
MaxConnIdleTime: 5 * time.Minute,
ConnectTimeout:  10 * time.Second,
}
}

// Client es el cliente PostgreSQL
type Client struct {
pool *pgxpool.Pool
cfg  Config
}

// New crea un nuevo cliente PostgreSQL
func New(ctx context.Context, cfg Config) (*Client, error) {
poolCfg, err := pgxpool.ParseConfig(cfg.URL)
if err != nil {
return nil, fmt.Errorf("failed to parse database URL: %w", err)
}

// Configurar pool
poolCfg.MaxConns = cfg.MaxConns
poolCfg.MinConns = cfg.MinConns
poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

// Crear pool
pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
if err != nil {
return nil, fmt.Errorf("failed to create pool: %w", err)
}

// Verificar conexión
pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

if err := pool.Ping(pingCtx); err != nil {
pool.Close()
return nil, fmt.Errorf("failed to ping database: %w", err)
}

return &Client{pool: pool, cfg: cfg}, nil
}

// Pool devuelve el pool de conexiones
func (c *Client) Pool() *pgxpool.Pool {
return c.pool
}

// Ping verifica la conexión
func (c *Client) Ping(ctx context.Context) error {
return c.pool.Ping(ctx)
}

// Close cierra el pool
func (c *Client) Close() {
c.pool.Close()
}

// Exec ejecuta una query sin retorno
func (c *Client) Exec(ctx context.Context, sql string, args ...interface{}) (int64, error) {
tag, err := c.pool.Exec(ctx, sql, args...)
if err != nil {
return 0, err
}
return tag.RowsAffected(), nil
}

// QueryRow ejecuta una query que devuelve una fila
func (c *Client) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
return c.pool.QueryRow(ctx, sql, args...)
}

// Query ejecuta una query que devuelve múltiples filas
func (c *Client) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
return c.pool.Query(ctx, sql, args...)
}

// BeginTx comienza una transacción
func (c *Client) BeginTx(ctx context.Context) (pgx.Tx, error) {
return c.pool.Begin(ctx)
}

// WithTx ejecuta una función dentro de una transacción
func (c *Client) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
tx, err := c.pool.Begin(ctx)
if err != nil {
return err
}
defer tx.Rollback(ctx)

if err := fn(tx); err != nil {
return err
}

return tx.Commit(ctx)
}

// Stats devuelve estadísticas del pool
func (c *Client) Stats() map[string]interface{} {
stat := c.pool.Stat()
return map[string]interface{}{
"total_conns":    stat.TotalConns(),
"idle_conns":     stat.IdleConns(),
"acquired_conns": stat.AcquiredConns(),
"max_conns":      stat.MaxConns(),
}
}

// HealthCheck verifica la salud de la conexión
func (c *Client) HealthCheck(ctx context.Context) error {
var result int
err := c.pool.QueryRow(ctx, "SELECT 1").Scan(&result)
if err != nil {
return fmt.Errorf("health check failed: %w", err)
}
if result != 1 {
return fmt.Errorf("unexpected health check result: %d", result)
}
return nil
}
