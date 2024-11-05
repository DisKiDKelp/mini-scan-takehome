package db

import (
	"context"
	"database/sql"
	"log"
	"os"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func NewPostgresConnection() (*DB, error) {
    host := os.Getenv("POSTGRES_HOST")
    user := os.Getenv("POSTGRES_USER")
    password := os.Getenv("POSTGRES_PASSWORD")
    dbname := os.Getenv("POSTGRES_DB")

    connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", host, user, password, dbname)
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
    }

    // Initialize tables if needed
    if _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS ip_addresses (
            id SERIAL PRIMARY KEY,
            ip TEXT,
            port INTEGER,
            service TEXT,
            last_updated TIMESTAMP,
            lock_id TEXT,
            UNIQUE (ip, port)
        );
        CREATE TABLE IF NOT EXISTS messages (
            id SERIAL PRIMARY KEY,
            ip_address_id INTEGER REFERENCES ip_addresses(id) ON DELETE CASCADE,
            message_content TEXT,
            service TEXT,
            insertion_time TIMESTAMP,
            response TEXT
        );
    `); err != nil {
        return nil, fmt.Errorf("failed to create tables: %w", err)
    }

    return &DB{db}, nil
}

func (db *DB) AcquireLock(ctx context.Context, ip string, port int) (string, error) {
	lockID := uuid.New().String()

	// Step 1: Check if the IP/port combination exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM ip_addresses WHERE ip = $1 AND port = $2)`
	err := db.QueryRowContext(ctx, checkQuery, ip, port).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("failed to check if IP/port exists: %w", err)
	}

	// Step 2: Insert a new row if the IP/port doesn’t exist
	if !exists {
		insertQuery := `
			INSERT INTO ip_addresses (ip, port, service, last_updated, response, lock_id)
			VALUES ($1, $2, '', NOW(), '', $3)
		`
		_, err := db.ExecContext(ctx, insertQuery, ip, port, lockID)
		if err != nil {
			return "", fmt.Errorf("failed to insert new IP/port record: %w", err)
		}
		log.Printf("Inserted new record for IP %s, Port %d", ip, port)
		return lockID, nil // Return immediately since we've acquired the lock with the new insert
	}

	// Step 3: Update the lock_id for existing rows if no lock is held
	updateQuery := `
		UPDATE ip_addresses
		SET lock_id = $1
		WHERE ip = $2
		AND port = $3
		AND (lock_id IS NULL OR lock_id = '')
	`
	result, err := db.ExecContext(ctx, updateQuery, lockID, ip, port)
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
	query := `
		UPDATE ip_addresses
		SET lock_id = NULL
		WHERE ip = $1 AND port = $2 AND lock_id = $3
	`
	_, err := db.ExecContext(ctx, query, ip, port, lockID)
	return err
}


// UpdateIPData upserts IP data in the database
func (db *DB) UpdateIPData(ctx context.Context, ip string, port int, service string, timestamp int64) error {
	query := `
		INSERT INTO ip_addresses (ip, port, service, last_updated)
		VALUES ($1, $2, $3, TO_TIMESTAMP($4))
		ON CONFLICT (ip, port) DO UPDATE SET
			service = EXCLUDED.service,
			last_updated = EXCLUDED.last_updated
	`
	_, err := db.ExecContext(ctx, query, ip, port, service, timestamp)
	return err
}

func (db *DB) GetOrInsertIPAddress(ctx context.Context, ip string, port int, service string, timestamp int64) (int, error) {
    var id int

    // Try to get the id of the existing ip/port
    query := `SELECT id FROM ip_addresses WHERE ip = $1 AND port = $2`
    err := db.QueryRowContext(ctx, query, ip, port).Scan(&id)
    if err == sql.ErrNoRows {
        // If not found, insert a new record
        insertQuery := `
            INSERT INTO ip_addresses (ip, port, service, last_updated)
            VALUES ($1, $2, $3, TO_TIMESTAMP($4))
            RETURNING id
        `
        err = db.QueryRowContext(ctx, insertQuery, ip, port, service, timestamp).Scan(&id)
        if err != nil {
            return 0, fmt.Errorf("failed to insert new IP/port record: %w", err)
        }

        log.Printf("Inserted new record for IP %s, Port %d", ip, port)

    } else if err != nil {
        return 0, fmt.Errorf("failed to retrieve IP/port id: %w", err)
    }

    return id, nil
}

func (db *DB) InsertMessage(ctx context.Context, ipAddressID int, messageContent, service, response string, insertionTime int64) error {
    query := `
        INSERT INTO messages (ip_address_id, message_content, service, response, insertion_time)
        VALUES ($1, $2, $3, $4, TO_TIMESTAMP($5))
    `

    _, err := db.ExecContext(ctx, query, ipAddressID, messageContent, service, response, insertionTime)
    if err != nil {
        return fmt.Errorf("failed to insert message: %w", err)
    }

    return nil
}