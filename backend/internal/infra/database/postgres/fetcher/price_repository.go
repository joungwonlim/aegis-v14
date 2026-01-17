package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// PriceRepository PostgreSQL 가격 저장소 (data.daily_prices)
type PriceRepository struct {
	pool *postgres.Pool
}

// NewPriceRepository 저장소 생성
func NewPriceRepository(pool *postgres.Pool) *PriceRepository {
	return &PriceRepository{pool: pool}
}

// Upsert 가격 저장
func (r *PriceRepository) Upsert(ctx context.Context, price *fetcher.DailyPrice) error {
	query := `
		INSERT INTO data.daily_prices
			(stock_code, trade_date, open_price, high_price, low_price, close_price, volume, trading_value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (stock_code, trade_date) DO UPDATE SET
			open_price = EXCLUDED.open_price,
			high_price = EXCLUDED.high_price,
			low_price = EXCLUDED.low_price,
			close_price = EXCLUDED.close_price,
			volume = EXCLUDED.volume,
			trading_value = EXCLUDED.trading_value
	`

	_, err := r.pool.Exec(ctx, query,
		price.StockCode, price.TradeDate,
		price.OpenPrice, price.HighPrice, price.LowPrice, price.ClosePrice,
		price.Volume, price.TradingValue,
	)
	if err != nil {
		return fmt.Errorf("upsert price: %w", err)
	}

	return nil
}

// UpsertBatch 가격 일괄 저장
func (r *PriceRepository) UpsertBatch(ctx context.Context, prices []*fetcher.DailyPrice) (int, error) {
	if len(prices) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO data.daily_prices
			(stock_code, trade_date, open_price, high_price, low_price, close_price, volume, trading_value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (stock_code, trade_date) DO UPDATE SET
			open_price = EXCLUDED.open_price,
			high_price = EXCLUDED.high_price,
			low_price = EXCLUDED.low_price,
			close_price = EXCLUDED.close_price,
			volume = EXCLUDED.volume,
			trading_value = EXCLUDED.trading_value
	`

	for _, price := range prices {
		batch.Queue(query,
			price.StockCode, price.TradeDate,
			price.OpenPrice, price.HighPrice, price.LowPrice, price.ClosePrice,
			price.Volume, price.TradingValue,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	count := 0
	for range prices {
		_, err := br.Exec()
		if err != nil {
			return count, fmt.Errorf("batch upsert price: %w", err)
		}
		count++
	}

	return count, nil
}

// GetByDate 특정 날짜 가격 조회
func (r *PriceRepository) GetByDate(ctx context.Context, stockCode string, date time.Time) (*fetcher.DailyPrice, error) {
	query := `
		SELECT stock_code, trade_date, open_price, high_price, low_price, close_price, volume, trading_value, created_at
		FROM data.daily_prices
		WHERE stock_code = $1 AND trade_date = $2
	`

	var price fetcher.DailyPrice
	err := r.pool.QueryRow(ctx, query, stockCode, date).Scan(
		&price.StockCode, &price.TradeDate,
		&price.OpenPrice, &price.HighPrice, &price.LowPrice, &price.ClosePrice,
		&price.Volume, &price.TradingValue, &price.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fetcher.ErrPriceNotFound
		}
		return nil, fmt.Errorf("get price: %w", err)
	}

	return &price, nil
}

// GetRange 기간별 가격 조회
func (r *PriceRepository) GetRange(ctx context.Context, stockCode string, from, to time.Time) ([]*fetcher.DailyPrice, error) {
	query := `
		SELECT stock_code, trade_date, open_price, high_price, low_price, close_price, volume, trading_value, created_at
		FROM data.daily_prices
		WHERE stock_code = $1 AND trade_date >= $2 AND trade_date <= $3
		ORDER BY trade_date DESC
	`

	rows, err := r.pool.Query(ctx, query, stockCode, from, to)
	if err != nil {
		return nil, fmt.Errorf("query prices: %w", err)
	}
	defer rows.Close()

	var prices []*fetcher.DailyPrice
	for rows.Next() {
		var price fetcher.DailyPrice
		if err := rows.Scan(
			&price.StockCode, &price.TradeDate,
			&price.OpenPrice, &price.HighPrice, &price.LowPrice, &price.ClosePrice,
			&price.Volume, &price.TradingValue, &price.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan price: %w", err)
		}
		prices = append(prices, &price)
	}

	return prices, rows.Err()
}

// GetLatest 최신 가격 조회
func (r *PriceRepository) GetLatest(ctx context.Context, stockCode string) (*fetcher.DailyPrice, error) {
	query := `
		SELECT stock_code, trade_date, open_price, high_price, low_price, close_price, volume, trading_value, created_at
		FROM data.daily_prices
		WHERE stock_code = $1
		ORDER BY trade_date DESC
		LIMIT 1
	`

	var price fetcher.DailyPrice
	err := r.pool.QueryRow(ctx, query, stockCode).Scan(
		&price.StockCode, &price.TradeDate,
		&price.OpenPrice, &price.HighPrice, &price.LowPrice, &price.ClosePrice,
		&price.Volume, &price.TradingValue, &price.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fetcher.ErrPriceNotFound
		}
		return nil, fmt.Errorf("get latest price: %w", err)
	}

	return &price, nil
}

// GetLatestN 최근 N일 가격 조회
func (r *PriceRepository) GetLatestN(ctx context.Context, stockCode string, n int) ([]*fetcher.DailyPrice, error) {
	query := `
		SELECT stock_code, trade_date, open_price, high_price, low_price, close_price, volume, trading_value, created_at
		FROM data.daily_prices
		WHERE stock_code = $1
		ORDER BY trade_date DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, stockCode, n)
	if err != nil {
		return nil, fmt.Errorf("query prices: %w", err)
	}
	defer rows.Close()

	var prices []*fetcher.DailyPrice
	for rows.Next() {
		var price fetcher.DailyPrice
		if err := rows.Scan(
			&price.StockCode, &price.TradeDate,
			&price.OpenPrice, &price.HighPrice, &price.LowPrice, &price.ClosePrice,
			&price.Volume, &price.TradingValue, &price.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan price: %w", err)
		}
		prices = append(prices, &price)
	}

	return prices, rows.Err()
}
