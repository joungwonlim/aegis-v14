package universe

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/universe"
)

// StatisticsReader implements universe.StatisticsReader
type StatisticsReader struct {
	db *pgxpool.Pool
}

// NewStatisticsReader creates a new statistics reader
func NewStatisticsReader(db *pgxpool.Pool) *StatisticsReader {
	return &StatisticsReader{db: db}
}

// GetStockStatistics retrieves stock statistics
func (r *StatisticsReader) GetStockStatistics(ctx context.Context, symbol string, days int) (*universe.StockStatistics, error) {
	// Calculate average volume and value over the last N days
	query := `
		SELECT
			symbol,
			COALESCE(AVG(CASE WHEN day_rank <= 5 THEN acc_volume END), 0) as avg_volume_5d,
			COALESCE(AVG(CASE WHEN day_rank <= 5 THEN price * acc_volume END), 0) as avg_value_5d,
			COALESCE(AVG(CASE WHEN day_rank <= 20 THEN acc_volume END), 0) as avg_volume_20d,
			COALESCE(AVG(CASE WHEN day_rank <= 20 THEN price * acc_volume END), 0) as avg_value_20d
		FROM (
			SELECT
				symbol,
				price,
				acc_volume,
				ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY updated_at DESC) as day_rank
			FROM market.prices
			WHERE symbol = $1
			  AND updated_at >= NOW() - INTERVAL '30 days'
		) ranked
		WHERE day_rank <= $2
		GROUP BY symbol
	`

	var stats universe.StockStatistics
	var avgVolume5D, avgValue5D, avgVolume20D, avgValue20D sql.NullFloat64

	err := r.db.QueryRow(ctx, query, symbol, days).Scan(
		&stats.Symbol,
		&avgVolume5D,
		&avgValue5D,
		&avgVolume20D,
		&avgValue20D,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// No data, return zeros
			return &universe.StockStatistics{
				Symbol:       symbol,
				AvgVolume5D:  0,
				AvgValue5D:   0,
				AvgVolume20D: 0,
				AvgValue20D:  0,
			}, nil
		}
		return nil, fmt.Errorf("query statistics: %w", err)
	}

	stats.AvgVolume5D = int64(avgVolume5D.Float64)
	stats.AvgValue5D = int64(avgValue5D.Float64)
	stats.AvgVolume20D = int64(avgVolume20D.Float64)
	stats.AvgValue20D = int64(avgValue20D.Float64)

	return &stats, nil
}

// GetBatchStatistics retrieves statistics for multiple symbols
func (r *StatisticsReader) GetBatchStatistics(ctx context.Context, symbols []string, days int) (map[string]*universe.StockStatistics, error) {
	if len(symbols) == 0 {
		return make(map[string]*universe.StockStatistics), nil
	}

	// Batch query for multiple symbols
	query := `
		SELECT
			symbol,
			COALESCE(AVG(CASE WHEN day_rank <= 5 THEN acc_volume END), 0) as avg_volume_5d,
			COALESCE(AVG(CASE WHEN day_rank <= 5 THEN price * acc_volume END), 0) as avg_value_5d,
			COALESCE(AVG(CASE WHEN day_rank <= 20 THEN acc_volume END), 0) as avg_volume_20d,
			COALESCE(AVG(CASE WHEN day_rank <= 20 THEN price * acc_volume END), 0) as avg_value_20d
		FROM (
			SELECT
				symbol,
				price,
				acc_volume,
				ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY updated_at DESC) as day_rank
			FROM market.prices
			WHERE symbol = ANY($1)
			  AND updated_at >= NOW() - INTERVAL '30 days'
		) ranked
		WHERE day_rank <= $2
		GROUP BY symbol
	`

	rows, err := r.db.Query(ctx, query, symbols, days)
	if err != nil {
		return nil, fmt.Errorf("query batch statistics: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*universe.StockStatistics)
	for rows.Next() {
		var stats universe.StockStatistics
		var avgVolume5D, avgValue5D, avgVolume20D, avgValue20D sql.NullFloat64

		err := rows.Scan(
			&stats.Symbol,
			&avgVolume5D,
			&avgValue5D,
			&avgVolume20D,
			&avgValue20D,
		)
		if err != nil {
			return nil, fmt.Errorf("scan statistics: %w", err)
		}

		stats.AvgVolume5D = int64(avgVolume5D.Float64)
		stats.AvgValue5D = int64(avgValue5D.Float64)
		stats.AvgVolume20D = int64(avgVolume20D.Float64)
		stats.AvgValue20D = int64(avgValue20D.Float64)

		result[stats.Symbol] = &stats
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	// Fill in missing symbols with zeros
	for _, symbol := range symbols {
		if _, exists := result[symbol]; !exists {
			result[symbol] = &universe.StockStatistics{
				Symbol:       symbol,
				AvgVolume5D:  0,
				AvgValue5D:   0,
				AvgVolume20D: 0,
				AvgValue20D:  0,
			}
		}
	}

	return result, nil
}
