package database

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDialect implements Dialect for SQLite
type SQLiteDialect struct{}

// NewSQLiteDialect creates a new SQLite dialect
func NewSQLiteDialect() *SQLiteDialect {
	return &SQLiteDialect{}
}

func (d *SQLiteDialect) DriverName() string {
	return "sqlite3"
}

func (d *SQLiteDialect) DSN(config DialectConfig) string {
	return config.Path
}

func (d *SQLiteDialect) RewriteQuery(query string) string {
	// SQLite uses ? placeholders, no rewrite needed
	return query
}

func (d *SQLiteDialect) SupportsLastInsertId() bool {
	return true
}

func (d *SQLiteDialect) ConfigureConnection(db *sql.DB) error {
	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return err
	}

	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return err
	}

	return nil
}

func (d *SQLiteDialect) MigrationsSubdir() string {
	return "sqlite"
}

func (d *SQLiteDialect) CreateMigrationsTableQuery() string {
	return `
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT UNIQUE NOT NULL,
			executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`
}
