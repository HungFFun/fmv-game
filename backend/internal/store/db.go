// Package store — data access layer trên SQLite (dev).
// Prod: chuyển sang Postgres (db/postgres-schema.sql); SQL ở đây viết chuẩn
// nên port gần như 1:1 (đổi placeholder ? → $n và driver).
package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // driver "sqlite", pure Go (không cần cgo)
)

//go:embed schema.sql
var schemaSQL string

// Store gói *sql.DB + các query có kiểu.
type Store struct {
	DB *sql.DB
}

// Open mở (hoặc tạo) DB tại path và chạy schema (idempotent).
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("tạo thư mục DB: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// SQLite + nhiều goroutine: 1 connection ghi là an toàn nhất.
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	} {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("pragma %q: %w", pragma, err)
		}
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("chạy schema: %w", err)
	}
	return &Store{DB: db}, nil
}

// OpenMemory mở DB in-memory — dùng cho test.
func OpenMemory() (*Store, error) {
	return Open(":memory:")
}

func (s *Store) Close() error { return s.DB.Close() }
