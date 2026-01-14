package universe

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HoldingReader implements universe.HoldingReader
type HoldingReader struct {
	db *pgxpool.Pool
}

// NewHoldingReader creates a new holding reader
func NewHoldingReader(db *pgxpool.Pool) *HoldingReader {
	return &HoldingReader{db: db}
}

// GetHoldings retrieves current holdings (qty > 0)
func (r *HoldingReader) GetHoldings(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT symbol
		FROM trade.positions
		WHERE qty > 0
		ORDER BY symbol
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query holdings: %w", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, fmt.Errorf("scan holding: %w", err)
		}
		symbols = append(symbols, symbol)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return symbols, nil
}
