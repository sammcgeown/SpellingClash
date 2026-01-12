package database

import (
	"database/sql"
)

// DBTX defines the database operations needed by repositories
// This interface is satisfied by both *DB and can be extended for transactions
type DBTX interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	ExecReturningID(query string, args ...interface{}) (int64, error)
	Begin() (*Tx, error)
	GetDialect() Dialect
}

// Tx wraps sql.Tx with dialect-aware methods
type Tx struct {
	*sql.Tx
	dialect Dialect
}

// Begin starts a new transaction
func (db *DB) Begin() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, dialect: db.Dialect}, nil
}

// GetDialect returns the database dialect
func (db *DB) GetDialect() Dialect {
	return db.Dialect
}

// Query executes a query with automatic placeholder rewriting
func (tx *Tx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return tx.Tx.Query(tx.dialect.RewriteQuery(query), args...)
}

// QueryRow executes a query that returns a single row with automatic placeholder rewriting
func (tx *Tx) QueryRow(query string, args ...interface{}) *sql.Row {
	return tx.Tx.QueryRow(tx.dialect.RewriteQuery(query), args...)
}

// Exec executes a query that doesn't return rows with automatic placeholder rewriting
func (tx *Tx) Exec(query string, args ...interface{}) (sql.Result, error) {
	return tx.Tx.Exec(tx.dialect.RewriteQuery(query), args...)
}

// ExecReturningID executes an INSERT and returns the new row's ID
func (tx *Tx) ExecReturningID(query string, args ...interface{}) (int64, error) {
	rewrittenQuery := tx.dialect.RewriteQuery(query)

	if tx.dialect.SupportsLastInsertId() {
		result, err := tx.Tx.Exec(rewrittenQuery, args...)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()
	}

	// PostgreSQL: append RETURNING id and use QueryRow
	rewrittenQuery = rewrittenQuery[:len(rewrittenQuery)-len(";")]
	if rewrittenQuery[len(rewrittenQuery)-1] != ')' {
		// Query doesn't end with ), so we need to be careful
	}
	rewrittenQuery += " RETURNING id"

	var id int64
	err := tx.Tx.QueryRow(rewrittenQuery, args...).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetDialect returns the transaction's dialect
func (tx *Tx) GetDialect() Dialect {
	return tx.dialect
}

// Commit commits the transaction
func (tx *Tx) Commit() error {
	return tx.Tx.Commit()
}

// Rollback aborts the transaction
func (tx *Tx) Rollback() error {
	return tx.Tx.Rollback()
}
