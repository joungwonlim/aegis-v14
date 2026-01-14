package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	"github.com/wonny/aegis/v14/internal/infra/kis"
	"github.com/wonny/aegis/v14/internal/pkg/config"
)

func main() {
	// Setup pretty logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	log.Info().Msg("🚀 Starting KIS Holdings Sync...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Context
	ctx := context.Background()

	// Initialize database connection
	dbPool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer dbPool.Close()

	log.Info().Msg("✅ Database connected")

	// Initialize KIS client
	kisClient, err := kis.NewClientFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize KIS client")
	}

	log.Info().Msg("✅ KIS client initialized")

	// Create KIS Execution Adapter
	kisAdapter := kis.NewExecutionAdapter(kisClient)

	// Get account ID from environment
	accountID := os.Getenv("KIS_ACCOUNT_ID")
	if accountID == "" {
		// Try KIS_ACCOUNT_NO as fallback
		accountID = os.Getenv("KIS_ACCOUNT_NO")
	}
	if accountID == "" {
		log.Fatal().Msg("KIS_ACCOUNT_ID or KIS_ACCOUNT_NO environment variable is required")
	}

	log.Info().Str("account_id", accountID).Msg("📊 Fetching holdings from KIS...")

	// Fetch holdings from KIS
	kisHoldings, err := kisAdapter.GetHoldings(ctx, accountID)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get holdings from KIS")
	}

	log.Info().Int("count", len(kisHoldings)).Msg("✅ Fetched holdings from KIS")

	if len(kisHoldings) == 0 {
		log.Info().Msg("⚠️  No holdings found")
		return
	}

	// Initialize holding repository
	holdingRepo := postgres.NewHoldingRepository(dbPool.Pool)

	// Upsert holdings to database
	for _, kh := range kisHoldings {
		symbolName := ""
		if kh.Raw != nil {
			if name, ok := kh.Raw["symbol_name"].(string); ok {
				symbolName = name
			}
		}

		holding := &execution.Holding{
			AccountID:    kh.AccountID,
			Symbol:       kh.Symbol,
			Qty:          kh.Qty,
			AvgPrice:     kh.AvgPrice,
			CurrentPrice: kh.CurrentPrice,
			Pnl:          kh.Pnl,
			PnlPct:       kh.PnlPct,
			UpdatedTS:    time.Now(),
			Raw:          kh.Raw,
		}

		if err := holdingRepo.UpsertHolding(ctx, holding); err != nil {
			log.Error().
				Err(err).
				Str("symbol", kh.Symbol).
				Msg("Failed to upsert holding")
			continue
		}

		log.Info().
			Str("symbol", kh.Symbol).
			Str("name", symbolName).
			Int64("qty", kh.Qty).
			Str("avg_price", kh.AvgPrice.String()).
			Str("current_price", kh.CurrentPrice.String()).
			Str("pnl", kh.Pnl.String()).
			Float64("pnl_pct", kh.PnlPct).
			Msg("💾 Saved holding")
	}

	log.Info().Msg("✅ All holdings synced successfully!")
	fmt.Println("\n🌐 Holdings are now available at: http://localhost:3099")
}
