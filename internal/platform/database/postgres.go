package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"araya-business-hub/be/internal/platform/logging"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

// Connect initializes a connection pool to PostgreSQL (Supabase) with pool configurations.
func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL cannot be empty")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	// Safe connection pool limits for Supabase connection pooler / direct connection
	config.MaxConns = 25
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 15 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database pool: %w", err)
	}

	// Test ping connection
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database ping check failed: %w", err)
	}

	logging.Info("Connected to PostgreSQL database successfully", map[string]any{
		"host": config.ConnConfig.Host,
		"port": config.ConnConfig.Port,
		"db":   config.ConnConfig.Database,
	})

	return &DB{Pool: pool}, nil
}

// Ping checks if the database is alive and reachable within a short timeout.
func (db *DB) Ping(ctx context.Context) error {
	if db == nil || db.Pool == nil {
		return errors.New("database connection is not initialized")
	}

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return db.Pool.Ping(pingCtx)
}

// Close gracefully terminates all connections in the pool.
func (db *DB) Close() {
	if db != nil && db.Pool != nil {
		db.Pool.Close()
		logging.Info("PostgreSQL connection pool closed", nil)
	}
}
