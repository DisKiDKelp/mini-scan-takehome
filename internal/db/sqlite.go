package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

// NewSQLiteConnection sets up a new SQLite connection and initializes tables if needed
func NewSQLiteConnection(dataSource string) (*DB, error) {
	db, err := sql.Open("sqlite3", dataSource)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS Ip_addresses (
			ip TEXT,
			port INTEGER,
			service TEXT,
			last_updated DATETIME,
			response TEXT,
			lock_id TEXT,
			PRIMARY KEY (ip, port)
		);
		CREATE TABLE IF NOT EXISTS Messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT,
			port INTEGER,
			message_content TEXT,
			service TEXT,
			insertion_time DATETIME,
			FOREIGN KEY (ip, port) REFERENCES Ip_addresses(ip, port)
		);
	`); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

// AcquireLock tries to set a unique lock on an IP/port entry
func (db *DB) AcquireLock(ctx context.Context, ip string, port int) (string, error) {
	lockID := uuid.New().String()
	query := `UPDATE Ip_addresses SET lock_id = ? WHERE ip = ? AND port = ? AND (lock_id IS NULL OR lock_id = '')`
	result, err := db.ExecContext(ctx, query, lockID, ip, port)
	if err != nil {
		return "", fmt.Errorf("failed to acquire lock: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return "", fmt.Errorf("could not acquire lock for IP %s and port %d", ip, port)
	}
	return lockID, nil
}

// ReleaseLock clears the lock ID if it matches the one that acquired it
func (db *DB) ReleaseLock(ctx context.Context, ip string, port int, lockID string) error {
	query := `UPDATE Ip_addresses SET lock_id = NULL WHERE ip = ? AND port = ? AND lock_id = ?`
	_, err := db.ExecContext(ctx, query, ip, port, lockID)
	return err
}

// UpdateIPData upserts IP data in the database
func (db *DB) UpdateIPData(ctx context.Context, ip string, port int, service string, timestamp int64, response string) error {
	query := `
		INSERT INTO Ip_addresses (ip, port, service, last_updated, response)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(ip, port) DO UPDATE SET
			service = excluded.service,
			last_updated = excluded.last_updated,
			response = excluded.response
	`
	_, err := db.ExecContext(ctx, query, ip, port, service, timestamp, response)
	return err
}
