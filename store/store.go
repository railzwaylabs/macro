// Package store provides Macro's GORM-backed database runtime.
package store

import (
	"context"
	"database/sql"
	"fmt"

	"gorm.io/gorm"
)

type transactionKey struct{}

// Store owns a GORM database and its underlying connection pool.
type Store struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

// New opens a database using the configured driver and configures its pool.
func New(cfg Config, opts ...gorm.Option) (*Store, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	dialector, err := dialector(cfg)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(dialector, opts...)
	if err != nil {
		return nil, fmt.Errorf("open %s store: %w", cfg.Driver, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get %s connection pool: %w", cfg.Driver, err)
	}
	configurePool(sqlDB, cfg)

	return &Store{db: db, sqlDB: sqlDB}, nil
}

// DB returns the context-bound GORM handle. Inside Transaction it returns the
// active transaction rather than the root connection.
func (s *Store) DB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(transactionKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return s.db.WithContext(ctx)
}

// Ping verifies that the database is reachable.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping store: %w", err)
	}
	return nil
}

// Close closes the underlying connection pool.
func (s *Store) Close() error {
	if err := s.sqlDB.Close(); err != nil {
		return fmt.Errorf("close store: %w", err)
	}
	return nil
}

// Transaction executes fn atomically. Repositories must obtain their handle
// through DB(ctx) so every operation uses the active transaction.
func (s *Store) Transaction(ctx context.Context, fn func(context.Context) error) error {
	if fn == nil {
		return nil
	}
	if _, ok := ctx.Value(transactionKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, transactionKey{}, tx))
	})
	return NormalizeError(err)
}

func configurePool(db *sql.DB, cfg Config) {
	if cfg.MaxOpenConnections > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConnections)
	}
	if cfg.MaxIdleConnections > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConnections)
	}
	if cfg.ConnectionMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnectionMaxLifetime)
	}
	if cfg.ConnectionMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnectionMaxIdleTime)
	}
}
