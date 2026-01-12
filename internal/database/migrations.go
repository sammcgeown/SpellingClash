package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// RunMigrations executes all SQL migration files in the migrations directory
// It automatically selects the correct subdirectory based on the database dialect
func (db *DB) RunMigrations(migrationsPath string) error {
	// Create migrations table if it doesn't exist
	if err := db.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Determine the dialect-specific migrations path
	dialectMigrationsPath := filepath.Join(migrationsPath, db.Dialect.MigrationsSubdir())

	// Check if dialect-specific folder exists, fall back to base path for backwards compatibility
	if _, err := os.Stat(dialectMigrationsPath); os.IsNotExist(err) {
		// Fall back to base migrations path (for backwards compatibility with existing SQLite setups)
		dialectMigrationsPath = migrationsPath
	}

	// Get all migration files
	files, err := filepath.Glob(filepath.Join(dialectMigrationsPath, "*.sql"))
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

		// Execute SQL migration
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
	query := db.Dialect.CreateMigrationsTableQuery()
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
