package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// FetcherStatusHandler handles fetcher status endpoints
type FetcherStatusHandler struct {
	pool *pgxpool.Pool
}

// NewFetcherStatusHandler creates a new FetcherStatusHandler
func NewFetcherStatusHandler(pool *pgxpool.Pool) *FetcherStatusHandler {
	return &FetcherStatusHandler{
		pool: pool,
	}
}

// TableStats 테이블 통계
type TableStats struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Count       int64  `json:"count"`
	LastUpdate  string `json:"last_update"`
	Status      string `json:"status"` // active, stale
}

// GetTableStats handles GET /api/v1/fetcher/tables/stats
func (h *FetcherStatusHandler) GetTableStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query: Get table statistics from data schema
	query := `
		WITH partition_counts AS (
			-- daily_prices partitions
			SELECT
				'daily_prices' as base_table,
				SUM(n_live_tup) as total_count
			FROM pg_stat_user_tables
			WHERE schemaname = 'data'
				AND relname LIKE 'daily_prices_%'

			UNION ALL

			-- investor_flow partitions
			SELECT
				'investor_flow' as base_table,
				SUM(n_live_tup) as total_count
			FROM pg_stat_user_tables
			WHERE schemaname = 'data'
				AND relname LIKE 'investor_flow_%'
		),
		regular_tables AS (
			SELECT
				relname as table_name,
				n_live_tup as count
			FROM pg_stat_user_tables
			WHERE schemaname = 'data'
				AND relname IN ('stocks', 'fundamentals', 'market_cap', 'disclosures')
		)
		SELECT
			COALESCE(p.base_table, r.table_name) as table_name,
			COALESCE(p.total_count, r.count) as record_count
		FROM partition_counts p
		FULL OUTER JOIN regular_tables r ON false
		ORDER BY table_name
	`

	rows, err := h.pool.Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query table stats")
		http.Error(w, "Failed to get table stats", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Map table names to display names
	displayNames := map[string]string{
		"stocks":        "종목 마스터",
		"daily_prices":  "일봉 데이터",
		"investor_flow": "투자자별 수급",
		"fundamentals":  "재무 데이터",
		"market_cap":    "시가총액",
		"disclosures":   "DART 공시",
	}

	var stats []TableStats
	for rows.Next() {
		var tableName string
		var count int64
		if err := rows.Scan(&tableName, &count); err != nil {
			log.Error().Err(err).Msg("Failed to scan table stats")
			continue
		}

		displayName, ok := displayNames[tableName]
		if !ok {
			displayName = tableName
		}

		// Determine status based on count
		status := "active"
		if count == 0 {
			status = "stale"
		}

		stats = append(stats, TableStats{
			Name:        tableName,
			DisplayName: displayName,
			Count:       count,
			LastUpdate:  "", // Will be populated from actual data
			Status:      status,
		})
	}

	// Get last update times for each table
	for i := range stats {
		tableName := stats[i].Name
		var lastUpdate *string

		switch tableName {
		case "stocks":
			h.pool.QueryRow(ctx, "SELECT MAX(updated_at)::text FROM data.stocks").Scan(&lastUpdate)
		case "daily_prices":
			// Check all partitions
			h.pool.QueryRow(ctx, `
				SELECT MAX(created_at)::text FROM (
					SELECT MAX(created_at) as created_at FROM data.daily_prices_2026_h1
					UNION ALL
					SELECT MAX(created_at) as created_at FROM data.daily_prices_2025_h2
					UNION ALL
					SELECT MAX(created_at) as created_at FROM data.daily_prices_2025_h1
					UNION ALL
					SELECT MAX(created_at) as created_at FROM data.daily_prices_2024_h2
				) t
			`).Scan(&lastUpdate)
		case "investor_flow":
			// Check all partitions
			h.pool.QueryRow(ctx, `
				SELECT MAX(created_at)::text FROM (
					SELECT MAX(created_at) as created_at FROM data.investor_flow_2026_h1
					UNION ALL
					SELECT MAX(created_at) as created_at FROM data.investor_flow_2025_h2
					UNION ALL
					SELECT MAX(created_at) as created_at FROM data.investor_flow_2025_h1
					UNION ALL
					SELECT MAX(created_at) as created_at FROM data.investor_flow_2024_h2
				) t
			`).Scan(&lastUpdate)
		case "fundamentals":
			h.pool.QueryRow(ctx, "SELECT MAX(created_at)::text FROM data.fundamentals").Scan(&lastUpdate)
		case "market_cap":
			h.pool.QueryRow(ctx, "SELECT MAX(created_at)::text FROM data.market_cap").Scan(&lastUpdate)
		case "disclosures":
			h.pool.QueryRow(ctx, "SELECT MAX(created_at)::text FROM data.disclosures").Scan(&lastUpdate)
		}

		if lastUpdate != nil {
			stats[i].LastUpdate = *lastUpdate
		}
	}

	response := map[string]interface{}{
		"tables": stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
