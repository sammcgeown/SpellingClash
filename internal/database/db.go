package database

import (
	"database/sql"
	"fmt"
	"strings"

	"spellingclash/internal/config"
)

// DB wraps the database connection with dialect support
type DB struct {
	*sql.DB
	Dialect Dialect
}

// Initialize creates and configures the database connection using SQLite (backwards compatible)
func Initialize(dbPath string) (*DB, error) {
	dialect := NewSQLiteDialect()
	dialectConfig := DialectConfig{Path: dbPath}

	db, err := sql.Open(dialect.DriverName(), dialect.DSN(dialectConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Apply dialect-specific configuration
	if err := dialect.ConfigureConnection(db); err != nil {
		return nil, fmt.Errorf("failed to configure connection: %w", err)
	}

	return &DB{DB: db, Dialect: dialect}, nil
}

// InitializeWithConfig creates and configures the database connection based on config
func InitializeWithConfig(cfg *config.Config) (*DB, error) {
	var dialect Dialect
	var dialectConfig DialectConfig

	switch strings.ToLower(cfg.DatabaseType) {
	case "postgres", "postgresql":
		dialect = NewPostgresDialect()
		dialectConfig = DialectConfig{URL: cfg.DatabaseURL}
	case "mysql":
		dialect = NewMySQLDialect()
		dialectConfig = DialectConfig{URL: cfg.DatabaseURL}
	case "sqlite", "sqlite3", "":
		dialect = NewSQLiteDialect()
		dialectConfig = DialectConfig{Path: cfg.DatabasePath}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.DatabaseType)
	}

	db, err := sql.Open(dialect.DriverName(), dialect.DSN(dialectConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Apply dialect-specific configuration
	if err := dialect.ConfigureConnection(db); err != nil {
		return nil, fmt.Errorf("failed to configure connection: %w", err)
	}

	return &DB{DB: db, Dialect: dialect}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// Query executes a query with automatic placeholder rewriting
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.Query(db.Dialect.RewriteQuery(query), args...)
}

// QueryRow executes a query that returns a single row with automatic placeholder rewriting
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRow(db.Dialect.RewriteQuery(query), args...)
}

// Exec executes a query that doesn't return rows with automatic placeholder rewriting
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.DB.Exec(db.Dialect.RewriteQuery(query), args...)
}

// ExecReturningID executes an INSERT query and returns the new row's ID
// This handles the dialect difference between databases that support LastInsertId()
// and PostgreSQL which requires RETURNING clause
func (db *DB) ExecReturningID(query string, args ...interface{}) (int64, error) {
	rewrittenQuery := db.Dialect.RewriteQuery(query)

	if db.Dialect.SupportsLastInsertId() {
		result, err := db.DB.Exec(rewrittenQuery, args...)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()
	}

	// PostgreSQL: append RETURNING id and use QueryRow
	rewrittenQuery = strings.TrimSuffix(strings.TrimSpace(rewrittenQuery), ";")
	rewrittenQuery += " RETURNING id"

	var id int64
	err := db.DB.QueryRow(rewrittenQuery, args...).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}
