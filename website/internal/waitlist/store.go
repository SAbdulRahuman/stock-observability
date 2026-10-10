// Package waitlist stores waitlist sign-ups in SQLite and serves the sign-up endpoint.
package waitlist

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Entry is one sign-up.
type Entry struct {
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Interest  string    `json:"interest"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

// Store is the SQLite-backed list of sign-ups.
type Store struct {
	db *sql.DB
}

const schema = `CREATE TABLE IF NOT EXISTS waitlist (
	id         INTEGER PRIMARY KEY,
	email      TEXT NOT NULL UNIQUE COLLATE NOCASE,
	name       TEXT NOT NULL DEFAULT '',
	interest   TEXT NOT NULL DEFAULT '',
	message    TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
)`

// Open opens or creates the database at path.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	// One writer keeps SQLite free of SQLITE_BUSY; the load is a handful of sign-ups.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("open waitlist %s: %w", path, err)
	}
	return &Store{db: db}, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Add stores e and reports whether the email was new.
func (s *Store) Add(ctx context.Context, e Entry) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO waitlist (email, name, interest, message, created_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(email) DO NOTHING`,
		e.Email, e.Name, e.Interest, e.Message, e.CreatedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// List returns all sign-ups, oldest first.
func (s *Store) List(ctx context.Context) ([]Entry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT email, name, interest, message, created_at FROM waitlist ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		var created string
		if err := rows.Scan(&e.Email, &e.Name, &e.Interest, &e.Message, &created); err != nil {
			return nil, err
		}
		if e.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
