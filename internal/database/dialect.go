package database

import (
	"database/sql"
	"regexp"
	"strconv"
)

// Dialect defines the interface for database-specific operations
type Dialect interface {
	// DriverName returns the driver name for sql.Open
	DriverName() string

	// DSN returns the data source name for the connection
	DSN(config DialectConfig) string

	// RewriteQuery converts placeholder syntax if needed (e.g., ? to $1 for postgres)
	RewriteQuery(query string) string

	// SupportsLastInsertId returns true if the driver supports LastInsertId()
	SupportsLastInsertId() bool

	// ConfigureConnection applies any database-specific connection settings
	ConfigureConnection(db *sql.DB) error

	// MigrationsSubdir returns the subdirectory name for migrations (e.g., "sqlite", "postgres")
	MigrationsSubdir() string

	// CreateMigrationsTableQuery returns the SQL to create the migrations tracking table
	CreateMigrationsTableQuery() string

	// BoolValue returns the SQL representation of a boolean value
	BoolValue(b bool) string
}

// DialectConfig holds configuration for database connection
type DialectConfig struct {
	// For SQLite
	Path string

	// For PostgreSQL/MySQL
	URL string
}

// placeholderRegexp matches ? placeholders not inside quotes
var placeholderRegexp = regexp.MustCompile(`\?`)

// rewritePlaceholdersToNumbered converts ? placeholders to $1, $2, etc.
func rewritePlaceholdersToNumbered(query string) string {
	counter := 0
	return placeholderRegexp.ReplaceAllStringFunc(query, func(match string) string {
		counter++
		return "$" + strconv.Itoa(counter)
	})
}
