package postgres

import (
"context"
"embed"
"fmt"
"sort"
"strings"

"github.com/lecodev-26/sentinelflow/internal/logger"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migration representa una migración
type Migration struct {
Version int
Name    string
SQL     string
}

// LoadMigrations carga las migraciones embebidas
func LoadMigrations() ([]Migration, error) {
entries, err := migrationsFS.ReadDir("migrations")
if err != nil {
return nil, fmt.Errorf("failed to read migrations dir: %w", err)
}

var migrations []Migration
for _, entry := range entries {
if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
continue
}

content, err := migrationsFS.ReadFile("migrations/" + entry.Name())
if err != nil {
return nil, fmt.Errorf("failed to read %s: %w", entry.Name(), err)
}

// Formato: NNNN_name.sql
parts := strings.SplitN(entry.Name(), "_", 2)
if len(parts) != 2 {
continue
}

var version int
if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
continue
}

name := strings.TrimSuffix(parts[1], ".sql")

migrations = append(migrations, Migration{
Version: version,
Name:    name,
SQL:     string(content),
})
}

sort.Slice(migrations, func(i, j int) bool {
return migrations[i].Version < migrations[j].Version
})

return migrations, nil
}

// Migrate aplica todas las migraciones pendientes
func (c *Client) Migrate(ctx context.Context) error {
// Crear tabla de migraciones si no existe
if _, err := c.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
version INTEGER PRIMARY KEY,
applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
checksum TEXT
)
`); err != nil {
return fmt.Errorf("failed to create migrations table: %w", err)
}

// Cargar migraciones
migrations, err := LoadMigrations()
if err != nil {
return fmt.Errorf("failed to load migrations: %w", err)
}

// Obtener versiones ya aplicadas
applied := make(map[int]bool)
rows, err := c.Query(ctx, "SELECT version FROM schema_migrations")
if err != nil {
return fmt.Errorf("failed to query migrations: %w", err)
}
for rows.Next() {
var v int
if err := rows.Scan(&v); err != nil {
rows.Close()
return err
}
applied[v] = true
}
rows.Close()

// Aplicar pendientes
appliedCount := 0
for _, m := range migrations {
if applied[m.Version] {
continue
}

logger.Infof("📦 Applying migration %04d_%s", m.Version, m.Name)

if _, err := c.Exec(ctx, m.SQL); err != nil {
return fmt.Errorf("migration %d failed: %w", m.Version, err)
}

if _, err := c.Exec(ctx,
"INSERT INTO schema_migrations (version) VALUES ($1)",
m.Version,
); err != nil {
return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
}

appliedCount++
}

if appliedCount == 0 {
logger.Info("✅ No pending migrations")
} else {
logger.Infof("✅ Applied %d migration(s)", appliedCount)
}

return nil
}
