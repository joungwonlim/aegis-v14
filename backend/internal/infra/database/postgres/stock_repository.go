package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/wonny/aegis/v14/internal/domain/stock"
)

// StockRepository implements stock.Repository using PostgreSQL
type StockRepository struct {
	pool *Pool
}

// NewStockRepository creates a new StockRepository
func NewStockRepository(pool *Pool) *StockRepository {
	return &StockRepository{pool: pool}
}

// List returns paginated stocks with filters
func (r *StockRepository) List(ctx context.Context, filter stock.ListFilter) (*stock.ListResult, error) {
	// Build WHERE clause
	whereClauses := []string{}
	args := []interface{}{}
	argIndex := 1

	// Status filter
	if filter.Status != "ALL" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	// Market filter
	if filter.Market != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("market = $%d", argIndex))
		args = append(args, *filter.Market)
		argIndex++
	}

	// Tradable filter
	if filter.IsTradable != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_tradable = $%d", argIndex))
		args = append(args, *filter.IsTradable)
		argIndex++
	}

	// Search filter (name or symbol)
	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(LOWER(name) LIKE $%d OR symbol LIKE $%d)", argIndex, argIndex))
		args = append(args, searchPattern)
		argIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM market.stocks %s", whereClause)
	var totalCount int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count stocks: %w", err)
	}

	// Build ORDER BY clause
	orderByClause := fmt.Sprintf("ORDER BY %s %s", filter.Sort, strings.ToUpper(filter.Order))

	// Handle NULL values in market_cap sorting (put NULLs last)
	if filter.Sort == "market_cap" {
		if filter.Order == "asc" {
			orderByClause = "ORDER BY market_cap ASC NULLS LAST"
		} else {
			orderByClause = "ORDER BY market_cap DESC NULLS LAST"
		}
	}

	// Calculate offset
	offset := (filter.Page - 1) * filter.Limit

	// Build query
	query := fmt.Sprintf(`
		SELECT symbol, name, market, status,
		       listing_date, delisting_date, sector, industry, market_cap,
		       is_tradable, trade_halt_reason, created_ts, updated_ts
		FROM market.stocks
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderByClause, argIndex, argIndex+1)

	args = append(args, filter.Limit, offset)

	// Execute query
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query stocks: %w", err)
	}
	defer rows.Close()

	// Scan results
	stocks := []stock.Stock{}
	for rows.Next() {
		var s stock.Stock
		err := rows.Scan(
			&s.Symbol, &s.Name, &s.Market, &s.Status,
			&s.ListingDate, &s.DelistingDate, &s.Sector, &s.Industry, &s.MarketCap,
			&s.IsTradable, &s.TradeHaltReason, &s.CreatedTS, &s.UpdatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stock: %w", err)
		}
		stocks = append(stocks, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating stocks: %w", err)
	}

	return &stock.ListResult{
		Stocks:     stocks,
		TotalCount: totalCount,
		Page:       filter.Page,
		Limit:      filter.Limit,
	}, nil
}

// GetBySymbol returns a stock by symbol
func (r *StockRepository) GetBySymbol(ctx context.Context, symbol string) (*stock.Stock, error) {
	query := `
		SELECT symbol, name, market, status,
		       listing_date, delisting_date, sector, industry, market_cap,
		       is_tradable, trade_halt_reason, created_ts, updated_ts
		FROM market.stocks
		WHERE symbol = $1
	`

	var s stock.Stock
	err := r.pool.QueryRow(ctx, query, symbol).Scan(
		&s.Symbol, &s.Name, &s.Market, &s.Status,
		&s.ListingDate, &s.DelistingDate, &s.Sector, &s.Industry, &s.MarketCap,
		&s.IsTradable, &s.TradeHaltReason, &s.CreatedTS, &s.UpdatedTS,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, stock.ErrStockNotFound
		}
		return nil, fmt.Errorf("failed to get stock: %w", err)
	}

	return &s, nil
}
