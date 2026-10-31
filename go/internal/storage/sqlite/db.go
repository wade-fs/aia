package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/wade/aia/internal/storage/sqlite/migrations"
)

type DB struct {
	db *sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
	}

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", dbPath)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	conn.SetMaxOpenConns(1) // SQLite write serializer
	conn.SetMaxIdleConns(1)

	db := &DB{db: conn}
	if err := db.Migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func (d *DB) RawDB() *sql.DB {
	return d.db
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) Migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := d.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations: %w", err)
	}

	for _, m := range migrations.All() {
		var exists int
		if err := d.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations WHERE version = ?", m.Version).Scan(&exists); err != nil {
			return fmt.Errorf("failed to read migration state: %w", err)
		}
		if exists > 0 {
			continue
		}
		err := d.WithTx(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
				return err
			}
			now := time.Now().UTC().Format(time.RFC3339)
			_, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)", m.Version, now)
			return err
		})
		if err != nil {
			return fmt.Errorf("error executing migration %d (%s): %w", m.Version, m.Name, err)
		}
	}
	return nil
}

// SchemaVersion returns the highest applied migration version.
func (d *DB) SchemaVersion(ctx context.Context) (int, error) {
	var v sql.NullInt64
	if err := d.db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&v); err != nil {
		return 0, err
	}
	return int(v.Int64), nil
}

// Ping verifies the database connection is usable.
func (d *DB) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (d *DB) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rollback err: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
