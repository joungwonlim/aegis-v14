package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/pkg/config"
)

// Pool wraps pgxpool.Pool
type Pool struct {
	*pgxpool.Pool
}

// NewPool creates a new PostgreSQL connection pool
// SSOT: config.Database.URL에서만 연결 정보를 가져옴
func NewPool(ctx context.Context, cfg *config.Config) (*Pool, error) {
	log.Info().
		Str("host", cfg.Database.Host).
		Str("port", cfg.Database.Port).
		Str("database", cfg.Database.Name).
		Str("user", cfg.Database.User).
		Msg("Connecting to PostgreSQL...")

	// Parse config from DATABASE_URL (SSOT)
	poolConfig, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Set pool configuration
	poolConfig.MaxConns = cfg.Database.MaxConns
	poolConfig.MinConns = cfg.Database.MinConns
	poolConfig.MaxConnLifetime = cfg.Database.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.Database.MaxConnIdleTime

	// Connect
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Ping to verify connection
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().Msg("✅ PostgreSQL connected successfully")

	// 권한 자동 체크
	if err := checkPermissions(ctx, pool); err != nil {
		log.Warn().Err(err).Msg("Permission check failed, but continuing...")
	}

	return &Pool{Pool: pool}, nil
}

// checkPermissions checks if the user has necessary permissions
func checkPermissions(ctx context.Context, pool *pgxpool.Pool) error {
	log.Info().Msg("Checking database permissions...")

	// Check schema access
	schemas := []string{"market", "trade", "system"}
	for _, schema := range schemas {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT 1 FROM pg_namespace WHERE nspname = $1
			)
		`
		err := pool.QueryRow(ctx, query, schema).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check schema %s: %w", schema, err)
		}

		if !exists {
			log.Warn().
				Str("schema", schema).
				Msg("⚠️  Schema does not exist (will be created by migrations)")
		}
	}

	log.Info().Msg("✅ Database connection OK")
	return nil
}

// Close closes the connection pool
func (p *Pool) Close() {
	log.Info().Msg("Closing PostgreSQL connection pool...")
	p.Pool.Close()
}

// DB returns a *sql.DB compatible interface for legacy code
func (p *Pool) DB() *sql.DB {
	return stdlib.OpenDBFromPool(p.Pool)
}
