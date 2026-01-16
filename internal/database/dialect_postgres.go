package database

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

// PostgresDialect implements Dialect for PostgreSQL
type PostgresDialect struct{}

// NewPostgresDialect creates a new PostgreSQL dialect
func NewPostgresDialect() *PostgresDialect {
	return &PostgresDialect{}
}

func (d *PostgresDialect) DriverName() string {
	return "postgres"
}

func (d *PostgresDialect) DSN(config DialectConfig) string {
	return config.URL
}

func (d *PostgresDialect) RewriteQuery(query string) string {
	// PostgreSQL uses $1, $2, etc. instead of ?
	return rewritePlaceholdersToNumbered(query)
}

func (d *PostgresDialect) SupportsLastInsertId() bool {
	// PostgreSQL doesn't support LastInsertId(), needs RETURNING clause
	return false
}

func (d *PostgresDialect) ConfigureConnection(db *sql.DB) error {
	// Configure connection pool for PostgreSQL
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// PostgreSQL has foreign keys enabled by default, no pragma needed
	return nil
}

func (d *PostgresDialect) MigrationsSubdir() string {
	return "postgres"
}

func (d *PostgresDialect) CreateMigrationsTableQuery() string {
	return `
		CREATE TABLE IF NOT EXISTS migrations (
			id BIGSERIAL PRIMARY KEY,
			filename TEXT UNIQUE NOT NULL,
			executed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);
	`
}

func (d *PostgresDialect) BoolValue(b bool) string {
	if b {
		return "TRUE"
	}
	return "FALSE"
}
