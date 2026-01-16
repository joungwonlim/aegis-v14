package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v14/internal/domain/price"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	"github.com/wonny/aegis/v14/internal/infra/naver"
	"github.com/wonny/aegis/v14/internal/pkg/config"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	// Connect to database
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbPool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}
	defer dbPool.Close()

	// Get holdings
	holdingRepo := postgres.NewHoldingRepository(dbPool.Pool)
	holdings, err := holdingRepo.GetAllHoldings(ctx)
	if err != nil {
		fmt.Printf("Failed to get holdings: %v\n", err)
		return
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

	fmt.Printf("📊 Fetching prices for %d symbols from Naver...\n", len(symbols))
	fmt.Printf("Symbols: %v\n", symbols)

	// Create Naver client
	naverClient := naver.NewClient()

	// Fetch prices
	ticks, err := naverClient.GetCurrentPrices(ctx, symbols)
	if err != nil {
		fmt.Printf("Failed to fetch prices: %v\n", err)
		return
	}

	fmt.Printf("\n📥 Fetched %d ticks from Naver\n", len(ticks))

	// Save to database
	priceRepo := postgres.NewPriceRepository(dbPool.Pool)
	saved := 0
	for _, tick := range ticks {
		// Convert Tick to UpsertBestPriceInput
		input := price.UpsertBestPriceInput{
			Symbol:      tick.Symbol,
			BestPrice:   tick.LastPrice,
			BestSource:  tick.Source,
			BestTS:      tick.TS,
			ChangePrice: tick.ChangePrice,
			ChangeRate:  tick.ChangeRate,
			Volume:      tick.Volume,
			BidPrice:    tick.BidPrice,
			AskPrice:    tick.AskPrice,
			IsStale:     false, // Fresh data from Naver
		}

		if err := priceRepo.UpsertBestPrice(ctx, input); err != nil {
			fmt.Printf("  ⚠️ %s: Failed to save (%v)\n", tick.Symbol, err)
			continue
		}
		fmt.Printf("  ✅ %s: %s원\n", tick.Symbol, formatPrice(tick.LastPrice))
		saved++
	}

	fmt.Printf("\n✅ Saved %d/%d prices\n", saved, len(ticks))
}

func formatPrice(price int64) string {
	if price >= 1000 {
		return fmt.Sprintf("%d,%03d", price/1000, price%1000)
	}
	return fmt.Sprintf("%d", price)
}
