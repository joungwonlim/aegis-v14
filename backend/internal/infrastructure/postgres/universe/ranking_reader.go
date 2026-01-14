package universe

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RankingReader implements universe.RankingReader
type RankingReader struct {
	db *pgxpool.Pool
}

// NewRankingReader creates a new ranking reader
func NewRankingReader(db *pgxpool.Pool) *RankingReader {
	return &RankingReader{db: db}
}

// GetRanking retrieves ranking by category and market
func (r *RankingReader) GetRanking(ctx context.Context, category, market string, limit int) ([]string, error) {
	// Query latest snapshot for the given category and market
	query := `
		WITH latest_snapshot AS (
			SELECT snapshot_date, snapshot_time
			FROM ranking.naver
			WHERE category = $1 AND market = $2
			ORDER BY snapshot_date DESC, snapshot_time DESC
			LIMIT 1
		)
		SELECT rn.stock_code
		FROM ranking.naver rn
		JOIN latest_snapshot ls
		  ON rn.snapshot_date = ls.snapshot_date
		  AND rn.snapshot_time = ls.snapshot_time
		WHERE rn.category = $1 AND rn.market = $2
		ORDER BY rn.rank_position ASC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, category, market, limit)
	if err != nil {
		return nil, fmt.Errorf("query ranking: %w", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, fmt.Errorf("scan ranking: %w", err)
		}
		symbols = append(symbols, symbol)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return symbols, nil
}
