package fetcher

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// StockRepository PostgreSQL 종목 저장소 (data.stocks)
type StockRepository struct {
	pool *postgres.Pool
}

// NewStockRepository 저장소 생성
func NewStockRepository(pool *postgres.Pool) *StockRepository {
	return &StockRepository{pool: pool}
}

// Upsert 종목 저장 (있으면 업데이트, 없으면 생성)
func (r *StockRepository) Upsert(ctx context.Context, stock *fetcher.Stock) error {
	query := `
		INSERT INTO data.stocks (code, name, market, sector, listing_date, delisting_date, status, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name,
			market = EXCLUDED.market,
			sector = EXCLUDED.sector,
			listing_date = EXCLUDED.listing_date,
			delisting_date = EXCLUDED.delisting_date,
			status = EXCLUDED.status,
			updated_at = NOW()
	`

	_, err := r.pool.Exec(ctx, query,
		stock.Code, stock.Name, stock.Market, stock.Sector,
		stock.ListingDate, stock.DelistingDate, stock.Status,
	)
	if err != nil {
		return fmt.Errorf("upsert stock: %w", err)
	}

	return nil
}

// UpsertBatch 종목 일괄 저장
func (r *StockRepository) UpsertBatch(ctx context.Context, stocks []*fetcher.Stock) (int, error) {
	if len(stocks) == 0 {
		return 0, nil
	}

	// 배치 upsert
	batch := &pgx.Batch{}
	query := `
		INSERT INTO data.stocks (code, name, market, sector, listing_date, delisting_date, status, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name,
			market = EXCLUDED.market,
			sector = EXCLUDED.sector,
			listing_date = EXCLUDED.listing_date,
			delisting_date = EXCLUDED.delisting_date,
			status = EXCLUDED.status,
			updated_at = NOW()
	`

	for _, stock := range stocks {
		batch.Queue(query,
			stock.Code, stock.Name, stock.Market, stock.Sector,
			stock.ListingDate, stock.DelistingDate, stock.Status,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	count := 0
	for range stocks {
		_, err := br.Exec()
		if err != nil {
			return count, fmt.Errorf("batch upsert stock: %w", err)
		}
		count++
	}

	return count, nil
}

// GetByCode 종목 조회
func (r *StockRepository) GetByCode(ctx context.Context, code string) (*fetcher.Stock, error) {
	query := `
		SELECT code, name, market, sector, listing_date, delisting_date, status, created_at, updated_at
		FROM data.stocks
		WHERE code = $1
	`

	var stock fetcher.Stock
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&stock.Code, &stock.Name, &stock.Market, &stock.Sector,
		&stock.ListingDate, &stock.DelistingDate, &stock.Status,
		&stock.CreatedAt, &stock.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fetcher.ErrStockNotFound
		}
		return nil, fmt.Errorf("get stock: %w", err)
	}

	return &stock, nil
}

// GetByMarket 시장별 종목 조회
func (r *StockRepository) GetByMarket(ctx context.Context, market string) ([]*fetcher.Stock, error) {
	query := `
		SELECT code, name, market, sector, listing_date, delisting_date, status, created_at, updated_at
		FROM data.stocks
		WHERE market = $1 AND status = 'active'
		ORDER BY code
	`

	rows, err := r.pool.Query(ctx, query, market)
	if err != nil {
		return nil, fmt.Errorf("query stocks: %w", err)
	}
	defer rows.Close()

	var stocks []*fetcher.Stock
	for rows.Next() {
		var stock fetcher.Stock
		if err := rows.Scan(
			&stock.Code, &stock.Name, &stock.Market, &stock.Sector,
			&stock.ListingDate, &stock.DelistingDate, &stock.Status,
			&stock.CreatedAt, &stock.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stock: %w", err)
		}
		stocks = append(stocks, &stock)
	}

	return stocks, rows.Err()
}

// GetActive 활성 종목 조회
func (r *StockRepository) GetActive(ctx context.Context) ([]*fetcher.Stock, error) {
	query := `
		SELECT code, name, market, sector, listing_date, delisting_date, status, created_at, updated_at
		FROM data.stocks
		WHERE status = 'active'
		ORDER BY code
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query stocks: %w", err)
	}
	defer rows.Close()

	var stocks []*fetcher.Stock
	for rows.Next() {
		var stock fetcher.Stock
		if err := rows.Scan(
			&stock.Code, &stock.Name, &stock.Market, &stock.Sector,
			&stock.ListingDate, &stock.DelistingDate, &stock.Status,
			&stock.CreatedAt, &stock.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stock: %w", err)
		}
		stocks = append(stocks, &stock)
	}

	return stocks, rows.Err()
}

// List 종목 목록 조회 (필터링)
func (r *StockRepository) List(ctx context.Context, filter *fetcher.StockFilter) ([]*fetcher.Stock, error) {
	query := `
		SELECT code, name, market, sector, listing_date, delisting_date, status, created_at, updated_at
		FROM data.stocks
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter != nil {
		if filter.Market != "" && filter.Market != "ALL" {
			query += fmt.Sprintf(" AND market = $%d", argIndex)
			args = append(args, filter.Market)
			argIndex++
		}

		if filter.Status != "" && filter.Status != "all" {
			query += fmt.Sprintf(" AND status = $%d", argIndex)
			args = append(args, filter.Status)
			argIndex++
		}
	}

	query += " ORDER BY code"

	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, filter.Limit)
			argIndex++
		}

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query stocks: %w", err)
	}
	defer rows.Close()

	var stocks []*fetcher.Stock
	for rows.Next() {
		var stock fetcher.Stock
		if err := rows.Scan(
			&stock.Code, &stock.Name, &stock.Market, &stock.Sector,
			&stock.ListingDate, &stock.DelistingDate, &stock.Status,
			&stock.CreatedAt, &stock.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stock: %w", err)
		}
		stocks = append(stocks, &stock)
	}

	return stocks, rows.Err()
}

// Count 종목 수 조회
func (r *StockRepository) Count(ctx context.Context, filter *fetcher.StockFilter) (int, error) {
	query := `SELECT COUNT(*) FROM data.stocks WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filter != nil {
		if filter.Market != "" && filter.Market != "ALL" {
			query += fmt.Sprintf(" AND market = $%d", argIndex)
			args = append(args, filter.Market)
			argIndex++
		}

		if filter.Status != "" && filter.Status != "all" {
			query += fmt.Sprintf(" AND status = $%d", argIndex)
			args = append(args, filter.Status)
		}
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count stocks: %w", err)
	}

	return count, nil
}
