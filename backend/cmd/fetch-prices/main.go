package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/wonny/aegis/v14/internal/infra/kis"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	"github.com/wonny/aegis/v14/internal/pkg/config"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Connect to database
	ctx := context.Background()
	dbPool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Create KIS client
	kisClient, err := kis.NewClientFromEnv()
	if err != nil {
		fmt.Printf("Failed to create KIS client: %v\n", err)
		os.Exit(1)
	}

	// Get holdings
	holdingRepo := postgres.NewHoldingRepository(dbPool.Pool)
	holdings, err := holdingRepo.GetAllHoldings(ctx)
	if err != nil {
		fmt.Printf("Failed to get holdings: %v\n", err)
		os.Exit(1)
	}

	if len(holdings) == 0 {
		fmt.Println("No holdings found")
		return
	}

	// Extract symbols
	symbols := make([]string, 0, len(holdings))
	for _, h := range holdings {
		symbols = append(symbols, h.Symbol)
	}

	fmt.Printf("Fetching prices for %d symbols: %s\n", len(symbols), strings.Join(symbols, ", "))

	// Fetch prices from KIS
	restClient := kisClient.REST()
	ticks, err := restClient.GetCurrentPrices(ctx, symbols)
	if err != nil {
		fmt.Printf("Failed to fetch prices: %v\n", err)
		os.Exit(1)
	}

	// Save to database
	priceRepo := postgres.NewPriceRepository(dbPool.Pool)
	saved := 0
	for _, tick := range ticks {
		if err := priceRepo.SaveTick(ctx, tick); err != nil {
			fmt.Printf("Failed to save tick for %s: %v\n", tick.Symbol, err)
			continue
		}
		saved++
	}

	fmt.Printf("✅ Saved %d/%d prices\n", saved, len(ticks))

	// Print sample
	for i, tick := range ticks {
		if i >= 5 {
			break
		}
		fmt.Printf("  %s: %d (bid: %v, ask: %v)\n",
			tick.Symbol, tick.Price, tick.BidPrice, tick.AskPrice)
	}
}
