package database

import (
	"testing"
)

func TestDialectSQLite(t *testing.T) {
	dialect := NewSQLiteDialect()
	
	t.Run("DriverName", func(t *testing.T) {
		result := dialect.DriverName()
		expected := "sqlite3"
		if result != expected {
			t.Errorf("DriverName() = %v, want %v", result, expected)
		}
	})
	
	t.Run("SupportsLastInsertId", func(t *testing.T) {
		result := dialect.SupportsLastInsertId()
		if !result {
			t.Error("SupportsLastInsertId() should return true for SQLite")
		}
	})
	
	t.Run("MigrationsSubdir", func(t *testing.T) {
		result := dialect.MigrationsSubdir()
		expected := "sqlite"
		if result != expected {
			t.Errorf("MigrationsSubdir() = %v, want %v", result, expected)
		}
	})
}

func TestDialectPostgreSQL(t *testing.T) {
	dialect := NewPostgresDialect()
	
	t.Run("DriverName", func(t *testing.T) {
		result := dialect.DriverName()
		expected := "postgres"
		if result != expected {
			t.Errorf("DriverName() = %v, want %v", result, expected)
		}
	})
	
	t.Run("SupportsLastInsertId", func(t *testing.T) {
		result := dialect.SupportsLastInsertId()
		if result {
			t.Error("SupportsLastInsertId() should return false for PostgreSQL")
		}
	})
	
	t.Run("MigrationsSubdir", func(t *testing.T) {
		result := dialect.MigrationsSubdir()
		expected := "postgres"
		if result != expected {
			t.Errorf("MigrationsSubdir() = %v, want %v", result, expected)
		}
	})
}

func TestDialectMySQL(t *testing.T) {
	dialect := NewMySQLDialect()
	
	t.Run("DriverName", func(t *testing.T) {
		result := dialect.DriverName()
		expected := "mysql"
		if result != expected {
			t.Errorf("DriverName() = %v, want %v", result, expected)
		}
	})
	
	t.Run("SupportsLastInsertId", func(t *testing.T) {
		result := dialect.SupportsLastInsertId()
		if !result {
			t.Error("SupportsLastInsertId() should return true for MySQL")
		}
	})
	
	t.Run("MigrationsSubdir", func(t *testing.T) {
		result := dialect.MigrationsSubdir()
		expected := "mysql"
		if result != expected {
			t.Errorf("MigrationsSubdir() = %v, want %v", result, expected)
		}
	})
}

func TestRewriteQuery(t *testing.T) {
	tests := []struct {
		name     string
		dialect  Dialect
		query    string
		expected string
	}{
		{
			name:     "SQLite no change",
			dialect:  NewSQLiteDialect(),
			query:    "SELECT * FROM users WHERE id = ?",
			expected: "SELECT * FROM users WHERE id = ?",
		},
		{
			name:     "PostgreSQL single placeholder",
			dialect:  NewPostgresDialect(),
			query:    "SELECT * FROM users WHERE id = ?",
			expected: "SELECT * FROM users WHERE id = $1",
		},
		{
			name:     "PostgreSQL multiple placeholders",
			dialect:  NewPostgresDialect(),
			query:    "INSERT INTO users (name, email) VALUES (?, ?)",
			expected: "INSERT INTO users (name, email) VALUES ($1, $2)",
		},
		{
			name:     "MySQL no change",
			dialect:  NewMySQLDialect(),
			query:    "UPDATE users SET name = ?, email = ? WHERE id = ?",
			expected: "UPDATE users SET name = ?, email = ? WHERE id = ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dialect.RewriteQuery(tt.query)
			if result != tt.expected {
				t.Errorf("RewriteQuery() = %v, want %v", result, tt.expected)
			}
		})
	}
}
