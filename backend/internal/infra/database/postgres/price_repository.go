package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v14/internal/domain/price"
)

// PriceRepository implements price.PriceRepository using PostgreSQL
type PriceRepository struct {
	pool *pgxpool.Pool
}

// NewPriceRepository creates a new PriceRepository
func NewPriceRepository(pool *pgxpool.Pool) *PriceRepository {
	return &PriceRepository{pool: pool}
}

// ============================================================================
// Tick Repository
// ============================================================================

// InsertTick inserts a new price tick
func (r *PriceRepository) InsertTick(ctx context.Context, tick price.Tick) error {
	query := `
		INSERT INTO market.prices_ticks (
			symbol, source, last_price, change_price, change_rate, volume,
			bid_price, ask_price, bid_volume, ask_volume, ts
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err := r.pool.Exec(ctx, query,
		tick.Symbol,
		tick.Source,
		tick.LastPrice,
		tick.ChangePrice,
		tick.ChangeRate,
		tick.Volume,
		tick.BidPrice,
		tick.AskPrice,
		tick.BidVolume,
		tick.AskVolume,
		tick.TS,
	)

	if err != nil {
		return fmt.Errorf("%w: %v", price.ErrDatabaseInsert, err)
	}

	return nil
}

// GetLatestTicks returns latest ticks per source for a symbol
func (r *PriceRepository) GetLatestTicks(ctx context.Context, symbol string) ([]price.Tick, error) {
	query := `
		SELECT DISTINCT ON (source)
			id, symbol, source, last_price, change_price, change_rate, volume,
			bid_price, ask_price, bid_volume, ask_volume, ts, created_ts
		FROM market.prices_ticks
		WHERE symbol = $1
		ORDER BY source, ts DESC
	`

	rows, err := r.pool.Query(ctx, query, symbol)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
	}
	defer rows.Close()

	var ticks []price.Tick
	for rows.Next() {
		var t price.Tick
		err := rows.Scan(
			&t.ID,
			&t.Symbol,
			&t.Source,
			&t.LastPrice,
			&t.ChangePrice,
			&t.ChangeRate,
			&t.Volume,
			&t.BidPrice,
			&t.AskPrice,
			&t.BidVolume,
			&t.AskVolume,
			&t.TS,
			&t.CreatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
		}
		ticks = append(ticks, t)
	}

	return ticks, nil
}

// GetLatestTickBySource returns latest tick for a symbol from specific source
func (r *PriceRepository) GetLatestTickBySource(ctx context.Context, symbol string, source price.Source) (*price.Tick, error) {
	query := `
		SELECT
			id, symbol, source, last_price, change_price, change_rate, volume,
			bid_price, ask_price, bid_volume, ask_volume, ts, created_ts
		FROM market.prices_ticks
		WHERE symbol = $1 AND source = $2
		ORDER BY ts DESC
		LIMIT 1
	`

	var t price.Tick
	err := r.pool.QueryRow(ctx, query, symbol, source).Scan(
		&t.ID,
		&t.Symbol,
		&t.Source,
		&t.LastPrice,
		&t.ChangePrice,
		&t.ChangeRate,
		&t.Volume,
		&t.BidPrice,
		&t.AskPrice,
		&t.BidVolume,
		&t.AskVolume,
		&t.TS,
		&t.CreatedTS,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // Not found is not an error
		}
		return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
	}

	return &t, nil
}

// GetTicksInTimeRange returns ticks within time range
func (r *PriceRepository) GetTicksInTimeRange(ctx context.Context, symbol string, start, end time.Time) ([]price.Tick, error) {
	query := `
		SELECT
			id, symbol, source, last_price, change_price, change_rate, volume,
			bid_price, ask_price, bid_volume, ask_volume, ts, created_ts
		FROM market.prices_ticks
		WHERE symbol = $1 AND ts >= $2 AND ts <= $3
		ORDER BY ts DESC
	`

	rows, err := r.pool.Query(ctx, query, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
	}
	defer rows.Close()

	var ticks []price.Tick
	for rows.Next() {
		var t price.Tick
		err := rows.Scan(
			&t.ID,
			&t.Symbol,
			&t.Source,
			&t.LastPrice,
			&t.ChangePrice,
			&t.ChangeRate,
			&t.Volume,
			&t.BidPrice,
			&t.AskPrice,
			&t.BidVolume,
			&t.AskVolume,
			&t.TS,
			&t.CreatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
		}
		ticks = append(ticks, t)
	}

	return ticks, nil
}

// ============================================================================
// BestPrice Repository
// ============================================================================

// UpsertBestPrice upserts best price for a symbol
func (r *PriceRepository) UpsertBestPrice(ctx context.Context, input price.UpsertBestPriceInput) error {
	query := `
		INSERT INTO market.prices_best (
			symbol, best_price, best_source, best_ts,
			change_price, change_rate, volume,
			bid_price, ask_price, is_stale, updated_ts
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW()
		)
		ON CONFLICT (symbol) DO UPDATE SET
			best_price = EXCLUDED.best_price,
			best_source = EXCLUDED.best_source,
			best_ts = EXCLUDED.best_ts,
			change_price = EXCLUDED.change_price,
			change_rate = EXCLUDED.change_rate,
			volume = EXCLUDED.volume,
			bid_price = EXCLUDED.bid_price,
			ask_price = EXCLUDED.ask_price,
			is_stale = EXCLUDED.is_stale,
			updated_ts = NOW()
	`

	_, err := r.pool.Exec(ctx, query,
		input.Symbol,
		input.BestPrice,
		input.BestSource,
		input.BestTS,
		input.ChangePrice,
		input.ChangeRate,
		input.Volume,
		input.BidPrice,
		input.AskPrice,
		input.IsStale,
	)

	if err != nil {
		return fmt.Errorf("%w: %v", price.ErrDatabaseUpdate, err)
	}

	return nil
}

// GetBestPrice returns best price for a symbol
func (r *PriceRepository) GetBestPrice(ctx context.Context, symbol string) (*price.BestPrice, error) {
	query := `
		SELECT
			symbol, best_price, best_source, best_ts,
			change_price, change_rate, volume,
			bid_price, ask_price, is_stale, updated_ts
		FROM market.prices_best
		WHERE symbol = $1
	`

	var bp price.BestPrice
	err := r.pool.QueryRow(ctx, query, symbol).Scan(
		&bp.Symbol,
		&bp.BestPrice,
		&bp.BestSource,
		&bp.BestTS,
		&bp.ChangePrice,
		&bp.ChangeRate,
		&bp.Volume,
		&bp.BidPrice,
		&bp.AskPrice,
		&bp.IsStale,
		&bp.UpdatedTS,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, price.ErrBestPriceNotFound
		}
		return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
	}

	return &bp, nil
}

// GetBestPrices returns best prices for multiple symbols
func (r *PriceRepository) GetBestPrices(ctx context.Context, symbols []string) ([]price.BestPrice, error) {
	if len(symbols) == 0 {
		return []price.BestPrice{}, nil
	}

	query := `
		SELECT
			symbol, best_price, best_source, best_ts,
			change_price, change_rate, volume,
			bid_price, ask_price, is_stale, updated_ts
		FROM market.prices_best
		WHERE symbol = ANY($1)
		ORDER BY symbol
	`

	rows, err := r.pool.Query(ctx, query, symbols)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
	}
	defer rows.Close()

	var prices []price.BestPrice
	for rows.Next() {
		var bp price.BestPrice
		err := rows.Scan(
			&bp.Symbol,
			&bp.BestPrice,
			&bp.BestSource,
			&bp.BestTS,
			&bp.ChangePrice,
			&bp.ChangeRate,
			&bp.Volume,
			&bp.BidPrice,
			&bp.AskPrice,
			&bp.IsStale,
			&bp.UpdatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
		}
		prices = append(prices, bp)
	}

	return prices, nil
}

// GetAllBestPrices returns all best prices
func (r *PriceRepository) GetAllBestPrices(ctx context.Context) ([]price.BestPrice, error) {
	query := `
		SELECT
			symbol, best_price, best_source, best_ts,
			change_price, change_rate, volume,
			bid_price, ask_price, is_stale, updated_ts
		FROM market.prices_best
		ORDER BY symbol
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
	}
	defer rows.Close()

	var prices []price.BestPrice
	for rows.Next() {
		var bp price.BestPrice
		err := rows.Scan(
			&bp.Symbol,
			&bp.BestPrice,
			&bp.BestSource,
			&bp.BestTS,
			&bp.ChangePrice,
			&bp.ChangeRate,
			&bp.Volume,
			&bp.BidPrice,
			&bp.AskPrice,
			&bp.IsStale,
			&bp.UpdatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
		}
		prices = append(prices, bp)
	}

	return prices, nil
}

// GetStalePrices returns symbols with stale prices
func (r *PriceRepository) GetStalePrices(ctx context.Context) ([]price.BestPrice, error) {
	query := `
		SELECT
			symbol, best_price, best_source, best_ts,
			change_price, change_rate, volume,
			bid_price, ask_price, is_stale, updated_ts
		FROM market.prices_best
		WHERE is_stale = true
		ORDER BY updated_ts DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
	}
	defer rows.Close()

	var prices []price.BestPrice
	for rows.Next() {
		var bp price.BestPrice
		err := rows.Scan(
			&bp.Symbol,
			&bp.BestPrice,
			&bp.BestSource,
			&bp.BestTS,
			&bp.ChangePrice,
			&bp.ChangeRate,
			&bp.Volume,
			&bp.BidPrice,
			&bp.AskPrice,
			&bp.IsStale,
			&bp.UpdatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
		}
		prices = append(prices, bp)
	}

	return prices, nil
}

// ============================================================================
// Freshness Repository
// ============================================================================

// UpsertFreshness upserts freshness data
func (r *PriceRepository) UpsertFreshness(ctx context.Context, input price.UpsertFreshnessInput) error {
	query := `
		INSERT INTO market.freshness (
			symbol, source, last_ts, last_price,
			is_stale, staleness_ms, quality_score, updated_ts
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW()
		)
		ON CONFLICT (symbol, source) DO UPDATE SET
			last_ts = EXCLUDED.last_ts,
			last_price = EXCLUDED.last_price,
			is_stale = EXCLUDED.is_stale,
			staleness_ms = EXCLUDED.staleness_ms,
			quality_score = EXCLUDED.quality_score,
			updated_ts = NOW()
	`

	_, err := r.pool.Exec(ctx, query,
		input.Symbol,
		input.Source,
		input.LastTS,
		input.LastPrice,
		input.IsStale,
		input.StalenessMS,
		input.QualityScore,
	)

	if err != nil {
		return fmt.Errorf("%w: %v", price.ErrDatabaseUpdate, err)
	}

	return nil
}

// GetFreshness returns freshness for symbol and source
func (r *PriceRepository) GetFreshness(ctx context.Context, symbol string, source price.Source) (*price.Freshness, error) {
	query := `
		SELECT
			symbol, source, last_ts, last_price,
			is_stale, staleness_ms, quality_score, updated_ts
		FROM market.freshness
		WHERE symbol = $1 AND source = $2
	`

	var f price.Freshness
	err := r.pool.QueryRow(ctx, query, symbol, source).Scan(
		&f.Symbol,
		&f.Source,
		&f.LastTS,
		&f.LastPrice,
		&f.IsStale,
		&f.StalenessMS,
		&f.QualityScore,
		&f.UpdatedTS,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, price.ErrFreshnessNotFound
		}
		return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
	}

	return &f, nil
}

// GetFreshnessBySymbol returns all freshness data for a symbol
func (r *PriceRepository) GetFreshnessBySymbol(ctx context.Context, symbol string) ([]price.Freshness, error) {
	query := `
		SELECT
			symbol, source, last_ts, last_price,
			is_stale, staleness_ms, quality_score, updated_ts
		FROM market.freshness
		WHERE symbol = $1
		ORDER BY quality_score DESC
	`

	rows, err := r.pool.Query(ctx, query, symbol)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
	}
	defer rows.Close()

	var freshnesses []price.Freshness
	for rows.Next() {
		var f price.Freshness
		err := rows.Scan(
			&f.Symbol,
			&f.Source,
			&f.LastTS,
			&f.LastPrice,
			&f.IsStale,
			&f.StalenessMS,
			&f.QualityScore,
			&f.UpdatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
		}
		freshnesses = append(freshnesses, f)
	}

	return freshnesses, nil
}

// GetFreshSymbols returns symbols with fresh prices
func (r *PriceRepository) GetFreshSymbols(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT symbol
		FROM market.freshness
		WHERE is_stale = false
		ORDER BY symbol
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		err := rows.Scan(&symbol)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
		}
		symbols = append(symbols, symbol)
	}

	return symbols, nil
}

// GetStaleSymbols returns symbols with stale prices
func (r *PriceRepository) GetStaleSymbols(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT symbol
		FROM market.freshness
		WHERE is_stale = true
		ORDER BY symbol
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		err := rows.Scan(&symbol)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", price.ErrDatabaseQuery, err)
		}
		symbols = append(symbols, symbol)
	}

	return symbols, nil
}
