package universe

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WatchlistReader implements universe.WatchlistReader
type WatchlistReader struct {
	db *pgxpool.Pool
}

// NewWatchlistReader creates a new watchlist reader
func NewWatchlistReader(db *pgxpool.Pool) *WatchlistReader {
	return &WatchlistReader{db: db}
}

// GetWatchlist retrieves watchlist
func (r *WatchlistReader) GetWatchlist(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT symbol
		FROM portfolio.watchlist
		WHERE is_active = true
		ORDER BY symbol
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query watchlist: %w", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, fmt.Errorf("scan watchlist: %w", err)
		}
		symbols = append(symbols, symbol)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return symbols, nil
}
