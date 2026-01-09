package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"wordclash/internal/utils"
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

		// Handle special migrations
		if filename == "009_populate_kid_credentials.sql" {
			if err := db.populateKidCredentials(); err != nil {
				return fmt.Errorf("failed to populate kid credentials: %w", err)
			}
		} else {
			// Execute standard SQL migration
			if err := db.executeMigration(string(content)); err != nil {
				return fmt.Errorf("failed to execute migration %s: %w", filename, err)
			}
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

// populateKidCredentials generates username and password for existing kids without credentials
func (db *DB) populateKidCredentials() error {
	// Get all kids without credentials
	rows, err := db.Query("SELECT id FROM kids WHERE username IS NULL OR username = '' OR password IS NULL OR password = ''")
	if err != nil {
		return fmt.Errorf("failed to query kids: %w", err)
	}
	defer rows.Close()

	var kidIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan kid ID: %w", err)
		}
		kidIDs = append(kidIDs, id)
	}

	if len(kidIDs) == 0 {
		fmt.Println("No kids need credential population")
		return nil
	}

	fmt.Printf("Populating credentials for %d kid(s)...\n", len(kidIDs))

	// Track used usernames to ensure uniqueness
	usedUsernames := make(map[string]bool)

	// Get existing usernames
	existingRows, err := db.Query("SELECT username FROM kids WHERE username IS NOT NULL AND username != ''")
	if err != nil {
		return fmt.Errorf("failed to query existing usernames: %w", err)
	}
	defer existingRows.Close()

	for existingRows.Next() {
		var username string
		if err := existingRows.Scan(&username); err != nil {
			return fmt.Errorf("failed to scan username: %w", err)
		}
		usedUsernames[username] = true
	}

	// Generate credentials for each kid
	for _, kidID := range kidIDs {
		// Generate unique username
		var username string
		maxRetries := 100
		for i := 0; i < maxRetries; i++ {
			username, err = utils.GenerateKidUsername()
			if err != nil {
				return fmt.Errorf("failed to generate username: %w", err)
			}
			if !usedUsernames[username] {
				usedUsernames[username] = true
				break
			}
		}

		// Generate password
		password, err := utils.GenerateKidPassword()
		if err != nil {
			return fmt.Errorf("failed to generate password: %w", err)
		}

		// Update kid with credentials
		_, err = db.Exec("UPDATE kids SET username = ?, password = ? WHERE id = ?", username, password, kidID)
		if err != nil {
			return fmt.Errorf("failed to update kid %d credentials: %w", kidID, err)
		}

		fmt.Printf("Generated credentials for kid ID %d: username=%s, password=%s\n", kidID, username, password)
	}

	return nil
}
