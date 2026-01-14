package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	"github.com/wonny/aegis/v14/internal/pkg/config"
)

func TestNewPool(t *testing.T) {
	// Skip if no database available
	t.Skip("Integration test - requires PostgreSQL")

	ctx := context.Background()

	cfg, err := config.Load()
	assert.NoError(t, err)

	pool, err := postgres.NewPool(ctx, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, pool)

	defer pool.Close()

	// Test ping
	err = pool.Ping(ctx)
	assert.NoError(t, err)
}

func TestPool_Health(t *testing.T) {
	// Skip if no database available
	t.Skip("Integration test - requires PostgreSQL")

	ctx := context.Background()

	cfg, err := config.Load()
	assert.NoError(t, err)

	pool, err := postgres.NewPool(ctx, cfg)
	assert.NoError(t, err)
	defer pool.Close()

	// Test health check
	health := pool.Health(ctx)
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health.Status)
	assert.Greater(t, health.MaxConns, int32(0))
}
