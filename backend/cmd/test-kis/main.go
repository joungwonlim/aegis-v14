package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/wonny/aegis/v14/internal/infra/kis"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Create KIS client
	client, err := kis.NewClientFromEnv()
	if err != nil {
		log.Fatalf("Failed to create KIS client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test 1: Get access token
	fmt.Println("========================================")
	fmt.Println("Test 1: Get Access Token")
	fmt.Println("========================================")
	token, err := client.Auth.GetAccessToken(ctx)
	if err != nil {
		log.Fatalf("Failed to get access token: %v", err)
	}
	fmt.Printf("✅ Access token obtained: %s...\n", token[:20])
	fmt.Println()

	// Test 2: Get current price for 삼성전자
	fmt.Println("========================================")
	fmt.Println("Test 2: Get Current Price (삼성전자)")
	fmt.Println("========================================")
	symbol := "005930" // 삼성전자
	tick, err := client.REST.GetCurrentPrice(ctx, symbol)
	if err != nil {
		log.Fatalf("Failed to get current price: %v", err)
	}

	fmt.Printf("Symbol: %s\n", tick.Symbol)
	fmt.Printf("Source: %s\n", tick.Source)
	fmt.Printf("Last Price: %d 원\n", tick.LastPrice)
	if tick.ChangePrice != nil {
		fmt.Printf("Change Price: %+d 원\n", *tick.ChangePrice)
	}
	if tick.ChangeRate != nil {
		fmt.Printf("Change Rate: %+.2f%%\n", *tick.ChangeRate)
	}
	if tick.Volume != nil {
		fmt.Printf("Volume: %d\n", *tick.Volume)
	}
	if tick.BidPrice != nil && tick.AskPrice != nil {
		fmt.Printf("Bid/Ask: %d / %d\n", *tick.BidPrice, *tick.AskPrice)
	}
	fmt.Printf("Timestamp: %s\n", tick.TS.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// Test 3: Get multiple prices
	fmt.Println("========================================")
	fmt.Println("Test 3: Get Multiple Prices")
	fmt.Println("========================================")
	symbols := []string{
		"005930", // 삼성전자
		"000660", // SK하이닉스
		"035420", // NAVER
	}

	ticks, err := client.REST.GetCurrentPrices(ctx, symbols)
	if err != nil {
		log.Fatalf("Failed to get current prices: %v", err)
	}

	fmt.Printf("Fetched %d prices:\n", len(ticks))
	for i, t := range ticks {
		changeStr := ""
		if t.ChangePrice != nil {
			changeStr = fmt.Sprintf("%+d", *t.ChangePrice)
		}
		fmt.Printf("%d. %s: %d원 (%s)\n", i+1, t.Symbol, t.LastPrice, changeStr)
	}
	fmt.Println()

	fmt.Println("========================================")
	fmt.Println("✅ All tests passed!")
	fmt.Println("========================================")

	os.Exit(0)
}
