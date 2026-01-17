package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// FundamentalsRepository PostgreSQL 재무 저장소 (data.fundamentals)
type FundamentalsRepository struct {
	pool *postgres.Pool
}

// NewFundamentalsRepository 저장소 생성
func NewFundamentalsRepository(pool *postgres.Pool) *FundamentalsRepository {
	return &FundamentalsRepository{pool: pool}
}

// Upsert 재무 저장
func (r *FundamentalsRepository) Upsert(ctx context.Context, fund *fetcher.Fundamentals) error {
	query := `
		INSERT INTO data.fundamentals
			(stock_code, report_date, per, pbr, psr, roe, debt_ratio, revenue, operating_profit, net_profit, eps, bps, dps)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (stock_code, report_date) DO UPDATE SET
			per = EXCLUDED.per,
			pbr = EXCLUDED.pbr,
			psr = EXCLUDED.psr,
			roe = EXCLUDED.roe,
			debt_ratio = EXCLUDED.debt_ratio,
			revenue = EXCLUDED.revenue,
			operating_profit = EXCLUDED.operating_profit,
			net_profit = EXCLUDED.net_profit,
			eps = EXCLUDED.eps,
			bps = EXCLUDED.bps,
			dps = EXCLUDED.dps
	`

	_, err := r.pool.Exec(ctx, query,
		fund.StockCode, fund.ReportDate,
		fund.PER, fund.PBR, fund.PSR, fund.ROE, fund.DebtRatio,
		fund.Revenue, fund.OperatingProfit, fund.NetProfit,
		fund.EPS, fund.BPS, fund.DPS,
	)
	if err != nil {
		return fmt.Errorf("upsert fundamentals: %w", err)
	}

	return nil
}

// UpsertBatch 재무 일괄 저장
func (r *FundamentalsRepository) UpsertBatch(ctx context.Context, funds []*fetcher.Fundamentals) (int, error) {
	if len(funds) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO data.fundamentals
			(stock_code, report_date, per, pbr, psr, roe, debt_ratio, revenue, operating_profit, net_profit, eps, bps, dps)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (stock_code, report_date) DO UPDATE SET
			per = EXCLUDED.per,
			pbr = EXCLUDED.pbr,
			psr = EXCLUDED.psr,
			roe = EXCLUDED.roe,
			debt_ratio = EXCLUDED.debt_ratio,
			revenue = EXCLUDED.revenue,
			operating_profit = EXCLUDED.operating_profit,
			net_profit = EXCLUDED.net_profit,
			eps = EXCLUDED.eps,
			bps = EXCLUDED.bps,
			dps = EXCLUDED.dps
	`

	for _, fund := range funds {
		batch.Queue(query,
			fund.StockCode, fund.ReportDate,
			fund.PER, fund.PBR, fund.PSR, fund.ROE, fund.DebtRatio,
			fund.Revenue, fund.OperatingProfit, fund.NetProfit,
			fund.EPS, fund.BPS, fund.DPS,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	count := 0
	for range funds {
		_, err := br.Exec()
		if err != nil {
			return count, fmt.Errorf("batch upsert fundamentals: %w", err)
		}
		count++
	}

	return count, nil
}

// GetLatest 최신 재무 조회
func (r *FundamentalsRepository) GetLatest(ctx context.Context, stockCode string) (*fetcher.Fundamentals, error) {
	query := `
		SELECT stock_code, report_date, per, pbr, psr, roe, debt_ratio, revenue, operating_profit, net_profit, eps, bps, dps, created_at
		FROM data.fundamentals
		WHERE stock_code = $1
		ORDER BY report_date DESC
		LIMIT 1
	`

	var fund fetcher.Fundamentals
	err := r.pool.QueryRow(ctx, query, stockCode).Scan(
		&fund.StockCode, &fund.ReportDate,
		&fund.PER, &fund.PBR, &fund.PSR, &fund.ROE, &fund.DebtRatio,
		&fund.Revenue, &fund.OperatingProfit, &fund.NetProfit,
		&fund.EPS, &fund.BPS, &fund.DPS,
		&fund.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fetcher.ErrFundamentalsNotFound
		}
		return nil, fmt.Errorf("get latest fundamentals: %w", err)
	}

	return &fund, nil
}

// GetByDate 특정 날짜 재무 조회
func (r *FundamentalsRepository) GetByDate(ctx context.Context, stockCode string, date time.Time) (*fetcher.Fundamentals, error) {
	query := `
		SELECT stock_code, report_date, per, pbr, psr, roe, debt_ratio, revenue, operating_profit, net_profit, eps, bps, dps, created_at
		FROM data.fundamentals
		WHERE stock_code = $1 AND report_date = $2
	`

	var fund fetcher.Fundamentals
	err := r.pool.QueryRow(ctx, query, stockCode, date).Scan(
		&fund.StockCode, &fund.ReportDate,
		&fund.PER, &fund.PBR, &fund.PSR, &fund.ROE, &fund.DebtRatio,
		&fund.Revenue, &fund.OperatingProfit, &fund.NetProfit,
		&fund.EPS, &fund.BPS, &fund.DPS,
		&fund.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fetcher.ErrFundamentalsNotFound
		}
		return nil, fmt.Errorf("get fundamentals: %w", err)
	}

	return &fund, nil
}

// GetRange 기간별 재무 조회
func (r *FundamentalsRepository) GetRange(ctx context.Context, stockCode string, from, to time.Time) ([]*fetcher.Fundamentals, error) {
	query := `
		SELECT stock_code, report_date, per, pbr, psr, roe, debt_ratio, revenue, operating_profit, net_profit, eps, bps, dps, created_at
		FROM data.fundamentals
		WHERE stock_code = $1 AND report_date >= $2 AND report_date <= $3
		ORDER BY report_date DESC
	`

	rows, err := r.pool.Query(ctx, query, stockCode, from, to)
	if err != nil {
		return nil, fmt.Errorf("query fundamentals: %w", err)
	}
	defer rows.Close()

	var funds []*fetcher.Fundamentals
	for rows.Next() {
		var fund fetcher.Fundamentals
		if err := rows.Scan(
			&fund.StockCode, &fund.ReportDate,
			&fund.PER, &fund.PBR, &fund.PSR, &fund.ROE, &fund.DebtRatio,
			&fund.Revenue, &fund.OperatingProfit, &fund.NetProfit,
			&fund.EPS, &fund.BPS, &fund.DPS,
			&fund.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan fundamentals: %w", err)
		}
		funds = append(funds, &fund)
	}

	return funds, rows.Err()
}
