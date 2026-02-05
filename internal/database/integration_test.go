package database

import (
	"context"
	"os"
	"testing"
)

// TestDatabaseIntegration tests the complete database lifecycle
func TestDatabaseIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with SQLite for integration testing
	dbPath := "test_integration.db"
	defer os.Remove(dbPath)

	// Test initialization
	db, err := Initialize(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Test connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Test that tables were created by migrations
	tables := []string{"users", "kids", "spelling_lists", "words", "practices", "practice_sessions"}

	for _, table := range tables {
		query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
		var name string
		err := db.QueryRowContext(ctx, query, table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s not found: %v", table, err)
		}
	}
}

// TestDatabaseTransactions tests transaction support
func TestDatabaseTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbPath := "test_transactions.db"
	defer os.Remove(dbPath)

	db, err := Initialize(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test successful transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Insert test data
	_, err = tx.ExecContext(ctx, "INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)",
		"testuser", "test@example.com", "hashedpass", "parent")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	// Commit
	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify data was inserted
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username = ?", "testuser").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query after commit: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 user, got %d", count)
	}

	// Test rollback
	tx2, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin second transaction: %v", err)
	}

	_, err = tx2.ExecContext(ctx, "INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)",
		"testuser2", "test2@example.com", "hashedpass", "parent")
	if err != nil {
		tx2.Rollback()
		t.Fatalf("Failed to insert in second transaction: %v", err)
	}

	// Rollback
	if err := tx2.Rollback(); err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Verify data was not inserted
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username = ?", "testuser2").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query after rollback: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 users after rollback, got %d", count)
	}
}

// TestConcurrentAccess tests concurrent database access
func TestConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbPath := "test_concurrent.db"
	defer os.Remove(dbPath)

	db, err := Initialize(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create test data
	_, err = db.ExecContext(ctx, "INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)",
		"concurrentuser", "concurrent@example.com", "hashedpass", "parent")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Run concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			var username string
			err := db.QueryRowContext(ctx, "SELECT username FROM users WHERE email = ?", "concurrent@example.com").Scan(&username)
			if err != nil {
				t.Errorf("Concurrent read failed: %v", err)
			}
			if username != "concurrentuser" {
				t.Errorf("Expected username 'concurrentuser', got '%s'", username)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
