package universe

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/universe"
)

// StockRepository implements universe.StockRepository
type StockRepository struct {
	db *pgxpool.Pool
}

// NewStockRepository creates a new stock repository
func NewStockRepository(db *pgxpool.Pool) *StockRepository {
	return &StockRepository{db: db}
}

// GetStockInfo retrieves stock information
func (r *StockRepository) GetStockInfo(ctx context.Context, symbol string) (*universe.StockInfo, error) {
	query := `
		SELECT
			stock_code,
			stock_name,
			market,
			COALESCE(sector, '기타') as sector,
			COALESCE(market_cap, 0) as market_cap,
			is_active,
			COALESCE(is_managed, false) as is_managed,
			COALESCE(is_suspended, false) as is_suspended,
			listed_date,
			updated_at
		FROM market.stocks
		WHERE stock_code = $1
	`

	var info universe.StockInfo
	err := r.db.QueryRow(ctx, query, symbol).Scan(
		&info.Symbol,
		&info.Name,
		&info.Market,
		&info.Sector,
		&info.MarketCap,
		&info.IsActive,
		&info.IsManaged,
		&info.IsSuspended,
		&info.ListedDate,
		&info.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("stock not found: %s", symbol)
		}
		return nil, fmt.Errorf("query stock info: %w", err)
	}

	return &info, nil
}

// GetActiveStocks retrieves all active stocks with filters
func (r *StockRepository) GetActiveStocks(ctx context.Context, criteria *universe.FilterCriteria) ([]*universe.StockInfo, error) {
	query := `
		SELECT
			stock_code,
			stock_name,
			market,
			COALESCE(sector, '기타') as sector,
			COALESCE(market_cap, 0) as market_cap,
			is_active,
			COALESCE(is_managed, false) as is_managed,
			COALESCE(is_suspended, false) as is_suspended,
			listed_date,
			updated_at
		FROM market.stocks
		WHERE is_active = true
		  AND ($1 = false OR is_managed = false)
		  AND ($2 = false OR is_suspended = false)
		  AND market_cap >= $3
		ORDER BY market_cap DESC
		LIMIT 1000
	`

	rows, err := r.db.Query(ctx, query,
		criteria.ExcludeManaged,
		criteria.ExcludeSuspended,
		criteria.MinMarketCap,
	)
	if err != nil {
		return nil, fmt.Errorf("query active stocks: %w", err)
	}
	defer rows.Close()

	var stocks []*universe.StockInfo
	for rows.Next() {
		var info universe.StockInfo
		err := rows.Scan(
			&info.Symbol,
			&info.Name,
			&info.Market,
			&info.Sector,
			&info.MarketCap,
			&info.IsActive,
			&info.IsManaged,
			&info.IsSuspended,
			&info.ListedDate,
			&info.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan stock: %w", err)
		}

		stocks = append(stocks, &info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return stocks, nil
}
