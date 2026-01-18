package universe

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StockInfoDetail represents company overview and additional info
type StockInfoDetail struct {
	ID               int64      `json:"id"`
	Symbol           string     `json:"symbol"`
	SymbolName       *string    `json:"symbol_name,omitempty"`
	CompanyOverview  *string    `json:"company_overview,omitempty"`
	OverviewSource   *string    `json:"overview_source,omitempty"`
	OverviewUpdatedAt *time.Time `json:"overview_updated_at,omitempty"`
	Sector           *string    `json:"sector,omitempty"`
	Industry         *string    `json:"industry,omitempty"`
	ListingDate      *time.Time `json:"listing_date,omitempty"`
	FiscalMonth      *int       `json:"fiscal_month,omitempty"`
	Homepage         *string    `json:"homepage,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// StockInfoRepository handles stocks_info table operations
type StockInfoRepository struct {
	db *pgxpool.Pool
}

// NewStockInfoRepository creates a new stock info repository
func NewStockInfoRepository(db *pgxpool.Pool) *StockInfoRepository {
	return &StockInfoRepository{db: db}
}

// GetBySymbol retrieves stock info by symbol
func (r *StockInfoRepository) GetBySymbol(ctx context.Context, symbol string) (*StockInfoDetail, error) {
	query := `
		SELECT
			id, symbol, symbol_name, company_overview, overview_source,
			overview_updated_at, sector, industry, listing_date,
			fiscal_month, homepage, created_at, updated_at
		FROM stocks_info
		WHERE symbol = $1
	`

	var info StockInfoDetail
	err := r.db.QueryRow(ctx, query, symbol).Scan(
		&info.ID,
		&info.Symbol,
		&info.SymbolName,
		&info.CompanyOverview,
		&info.OverviewSource,
		&info.OverviewUpdatedAt,
		&info.Sector,
		&info.Industry,
		&info.ListingDate,
		&info.FiscalMonth,
		&info.Homepage,
		&info.CreatedAt,
		&info.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found, return nil without error
		}
		return nil, err
	}

	return &info, nil
}

// UpsertCompanyOverview inserts or updates company overview for a symbol
func (r *StockInfoRepository) UpsertCompanyOverview(ctx context.Context, symbol, symbolName, overview, source string) error {
	query := `
		INSERT INTO stocks_info (symbol, symbol_name, company_overview, overview_source, overview_updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (symbol) DO UPDATE SET
			symbol_name = COALESCE(EXCLUDED.symbol_name, stocks_info.symbol_name),
			company_overview = EXCLUDED.company_overview,
			overview_source = EXCLUDED.overview_source,
			overview_updated_at = NOW(),
			updated_at = NOW()
	`

	_, err := r.db.Exec(ctx, query, symbol, symbolName, overview, source)
	return err
}

// HasCompanyOverview checks if a symbol has company overview
func (r *StockInfoRepository) HasCompanyOverview(ctx context.Context, symbol string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM stocks_info
			WHERE symbol = $1 AND company_overview IS NOT NULL AND company_overview != ''
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, symbol).Scan(&exists)
	return exists, err
}
