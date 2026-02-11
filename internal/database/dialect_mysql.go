package database

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLDialect implements Dialect for MySQL
type MySQLDialect struct{}

// NewMySQLDialect creates a new MySQL dialect
func NewMySQLDialect() *MySQLDialect {
	return &MySQLDialect{}
}

func (d *MySQLDialect) DriverName() string {
	return "mysql"
}

func (d *MySQLDialect) DSN(config DialectConfig) string {
	return config.URL
}

func (d *MySQLDialect) RewriteQuery(query string) string {
	// MySQL uses ? placeholders like SQLite, no rewrite needed
	return query
}

func (d *MySQLDialect) SupportsLastInsertId() bool {
	return true
}

func (d *MySQLDialect) ConfigureConnection(db *sql.DB) error {
	// Configure connection pool for MySQL
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Ensure foreign key checks are enabled
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS = 1;"); err != nil {
		return err
	}

	return nil
}

func (d *MySQLDialect) MigrationsSubdir() string {
	return "mysql"
}

func (d *MySQLDialect) CreateMigrationsTableQuery() string {
	return `
		CREATE TABLE IF NOT EXISTS migrations (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			filename VARCHAR(255) UNIQUE NOT NULL,
			executed_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6)
		);
	`
}

func (d *MySQLDialect) BoolValue(b bool) string {
	if b {
		return "TRUE"
	}
	return "FALSE"
}

func (d *MySQLDialect) UpsertSettings() string {
	return "INSERT INTO settings (`key`, `value`) VALUES (?, ?) " +
		"ON DUPLICATE KEY UPDATE `value` = VALUES(`value`), updated_at = CURRENT_TIMESTAMP"
}
