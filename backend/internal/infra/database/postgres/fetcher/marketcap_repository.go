package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// MarketCapRepository PostgreSQL 시가총액 저장소 (data.market_cap)
type MarketCapRepository struct {
	pool *postgres.Pool
}

// NewMarketCapRepository 저장소 생성
func NewMarketCapRepository(pool *postgres.Pool) *MarketCapRepository {
	return &MarketCapRepository{pool: pool}
}

// Upsert 시가총액 저장
func (r *MarketCapRepository) Upsert(ctx context.Context, mc *fetcher.MarketCap) error {
	query := `
		INSERT INTO data.market_cap (stock_code, trade_date, market_cap, shares_out, float_shares)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (stock_code, trade_date) DO UPDATE SET
			market_cap = EXCLUDED.market_cap,
			shares_out = EXCLUDED.shares_out,
			float_shares = EXCLUDED.float_shares
	`

	_, err := r.pool.Exec(ctx, query,
		mc.StockCode, mc.TradeDate, mc.MarketCap, mc.SharesOut, mc.FloatShares,
	)
	if err != nil {
		return fmt.Errorf("upsert market cap: %w", err)
	}

	return nil
}

// UpsertBatch 시가총액 일괄 저장
func (r *MarketCapRepository) UpsertBatch(ctx context.Context, mcs []*fetcher.MarketCap) (int, error) {
	if len(mcs) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO data.market_cap (stock_code, trade_date, market_cap, shares_out, float_shares)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (stock_code, trade_date) DO UPDATE SET
			market_cap = EXCLUDED.market_cap,
			shares_out = EXCLUDED.shares_out,
			float_shares = EXCLUDED.float_shares
	`

	for _, mc := range mcs {
		batch.Queue(query, mc.StockCode, mc.TradeDate, mc.MarketCap, mc.SharesOut, mc.FloatShares)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	count := 0
	for range mcs {
		_, err := br.Exec()
		if err != nil {
			return count, fmt.Errorf("batch upsert market cap: %w", err)
		}
		count++
	}

	return count, nil
}

// GetLatest 최신 시가총액 조회
func (r *MarketCapRepository) GetLatest(ctx context.Context, stockCode string) (*fetcher.MarketCap, error) {
	query := `
		SELECT stock_code, trade_date, market_cap, shares_out, float_shares, created_at
		FROM data.market_cap
		WHERE stock_code = $1
		ORDER BY trade_date DESC
		LIMIT 1
	`

	var mc fetcher.MarketCap
	err := r.pool.QueryRow(ctx, query, stockCode).Scan(
		&mc.StockCode, &mc.TradeDate, &mc.MarketCap, &mc.SharesOut, &mc.FloatShares, &mc.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fetcher.ErrMarketCapNotFound
		}
		return nil, fmt.Errorf("get latest market cap: %w", err)
	}

	return &mc, nil
}

// GetByDate 특정 날짜 시가총액 조회
func (r *MarketCapRepository) GetByDate(ctx context.Context, stockCode string, date time.Time) (*fetcher.MarketCap, error) {
	query := `
		SELECT stock_code, trade_date, market_cap, shares_out, float_shares, created_at
		FROM data.market_cap
		WHERE stock_code = $1 AND trade_date = $2
	`

	var mc fetcher.MarketCap
	err := r.pool.QueryRow(ctx, query, stockCode, date).Scan(
		&mc.StockCode, &mc.TradeDate, &mc.MarketCap, &mc.SharesOut, &mc.FloatShares, &mc.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fetcher.ErrMarketCapNotFound
		}
		return nil, fmt.Errorf("get market cap: %w", err)
	}

	return &mc, nil
}

// GetTopN 시가총액 상위 N개 조회
func (r *MarketCapRepository) GetTopN(ctx context.Context, date time.Time, n int) ([]*fetcher.MarketCap, error) {
	query := `
		SELECT stock_code, trade_date, market_cap, shares_out, float_shares, created_at
		FROM data.market_cap
		WHERE trade_date = $1
		ORDER BY market_cap DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, date, n)
	if err != nil {
		return nil, fmt.Errorf("query market caps: %w", err)
	}
	defer rows.Close()

	var mcs []*fetcher.MarketCap
	for rows.Next() {
		var mc fetcher.MarketCap
		if err := rows.Scan(
			&mc.StockCode, &mc.TradeDate, &mc.MarketCap, &mc.SharesOut, &mc.FloatShares, &mc.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan market cap: %w", err)
		}
		mcs = append(mcs, &mc)
	}

	return mcs, rows.Err()
}
