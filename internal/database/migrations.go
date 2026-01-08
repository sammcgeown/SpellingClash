package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// RunMigrations executes all SQL migration files in the migrations directory
func (db *DB) RunMigrations(migrationsPath string) error {
	// Create migrations table if it doesn't exist
	if err := db.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get all migration files
	files, err := filepath.Glob(filepath.Join(migrationsPath, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	// Sort files to ensure they run in order
	sort.Strings(files)

	// Run each migration
	for _, file := range files {
		filename := filepath.Base(file)

		// Check if migration has already been run
		hasRun, err := db.hasMigrationRun(filename)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if hasRun {
			continue
		}

		// Read migration file
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Execute migration
		if err := db.executeMigration(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		// Record migration as completed
		if err := db.recordMigration(filename); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		fmt.Printf("Migration completed: %s\n", filename)
	}

	return nil
}

// createMigrationsTable creates the table to track completed migrations
func (db *DB) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT UNIQUE NOT NULL,
			executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := db.Exec(query)
	return err
}

// hasMigrationRun checks if a migration has already been executed
func (db *DB) hasMigrationRun(filename string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM migrations WHERE filename = ?"
	err := db.QueryRow(query, filename).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// executeMigration runs the SQL statements in a migration
func (db *DB) executeMigration(content string) error {
	// Execute the entire migration file content
	// SQLite can handle multiple statements in one Exec call
	_, err := db.Exec(content)
	return err
}

// recordMigration marks a migration as completed
func (db *DB) recordMigration(filename string) error {
	query := "INSERT INTO migrations (filename) VALUES (?)"
	_, err := db.Exec(query, filename)
	return err
}
