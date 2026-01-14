package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	"github.com/wonny/aegis/v14/internal/pkg/config"
)

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	log.Info().Msg("🚀 Converting Holdings to Positions...")

	ctx := context.Background()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	// Connect to DB
	dbPool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to DB")
	}
	defer dbPool.Close()
	log.Info().Msg("✅ Database connected")

	// Get default exit profile ID
	defaultProfileID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	// Convert holdings to positions
	query := `
		INSERT INTO trade.positions (
			position_id,
			account_id,
			symbol,
			side,
			qty,
			avg_price,
			entry_ts,
			status,
			exit_mode,
			exit_profile_id,
			strategy_id,
			updated_ts,
			version
		)
		SELECT
			gen_random_uuid(),
			h.account_id,
			h.symbol,
			'LONG',
			h.qty,
			h.avg_price,
			NOW() - INTERVAL '30 days',  -- 기존 포지션으로 가정 (30일 전 진입)
			'OPEN',
			'ENABLED',
			$1,  -- default exit profile
			'EXISTING',
			NOW(),
			1
		FROM trade.holdings h
		WHERE h.qty > 0
		ON CONFLICT DO NOTHING
		RETURNING position_id, symbol, qty, avg_price
	`

	rows, err := dbPool.Pool.Query(ctx, query, defaultProfileID)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to convert holdings")
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var positionID uuid.UUID
		var symbol string
		var qty int64
		var avgPrice float64

		if err := rows.Scan(&positionID, &symbol, &qty, &avgPrice); err != nil {
			log.Error().Err(err).Msg("Failed to scan row")
			continue
		}

		// Create position state
		stateQuery := `
			INSERT INTO trade.position_states (position_id, phase, last_eval_ts, updated_ts)
			VALUES ($1, 'MONITORING', NOW(), NOW())
			ON CONFLICT (position_id) DO NOTHING
		`
		if _, err := dbPool.Pool.Exec(ctx, stateQuery, positionID); err != nil {
			log.Error().Err(err).Str("symbol", symbol).Msg("Failed to create position state")
			continue
		}

		log.Info().
			Str("symbol", symbol).
			Int64("qty", qty).
			Float64("avg_price", avgPrice).
			Str("position_id", positionID.String()).
			Msg("✅ Position created")
		count++
	}

	if err := rows.Err(); err != nil {
		log.Fatal().Err(err).Msg("Rows error")
	}

	log.Info().Int("count", count).Msg("✅ All holdings converted to positions")
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("  - Created: %d positions\n", count)
	fmt.Printf("  - Exit Profile: default\n")
	fmt.Printf("  - Exit Mode: ENABLED\n")
	fmt.Printf("  - Status: OPEN\n")
	fmt.Printf("\n✅ Ready for Exit Engine monitoring!\n")
}
