package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// FlowRepository PostgreSQL 수급 저장소 (data.investor_flow)
type FlowRepository struct {
	pool *postgres.Pool
}

// NewFlowRepository 저장소 생성
func NewFlowRepository(pool *postgres.Pool) *FlowRepository {
	return &FlowRepository{pool: pool}
}

// Upsert 수급 저장
func (r *FlowRepository) Upsert(ctx context.Context, flow *fetcher.InvestorFlow) error {
	query := `
		INSERT INTO data.investor_flow
			(stock_code, trade_date, foreign_net_qty, foreign_net_value, inst_net_qty, inst_net_value, indiv_net_qty, indiv_net_value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (stock_code, trade_date) DO UPDATE SET
			foreign_net_qty = EXCLUDED.foreign_net_qty,
			foreign_net_value = EXCLUDED.foreign_net_value,
			inst_net_qty = EXCLUDED.inst_net_qty,
			inst_net_value = EXCLUDED.inst_net_value,
			indiv_net_qty = EXCLUDED.indiv_net_qty,
			indiv_net_value = EXCLUDED.indiv_net_value
	`

	_, err := r.pool.Exec(ctx, query,
		flow.StockCode, flow.TradeDate,
		flow.ForeignNetQty, flow.ForeignNetValue,
		flow.InstNetQty, flow.InstNetValue,
		flow.IndivNetQty, flow.IndivNetValue,
	)
	if err != nil {
		return fmt.Errorf("upsert flow: %w", err)
	}

	return nil
}

// UpsertBatch 수급 일괄 저장
func (r *FlowRepository) UpsertBatch(ctx context.Context, flows []*fetcher.InvestorFlow) (int, error) {
	if len(flows) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO data.investor_flow
			(stock_code, trade_date, foreign_net_qty, foreign_net_value, inst_net_qty, inst_net_value, indiv_net_qty, indiv_net_value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (stock_code, trade_date) DO UPDATE SET
			foreign_net_qty = EXCLUDED.foreign_net_qty,
			foreign_net_value = EXCLUDED.foreign_net_value,
			inst_net_qty = EXCLUDED.inst_net_qty,
			inst_net_value = EXCLUDED.inst_net_value,
			indiv_net_qty = EXCLUDED.indiv_net_qty,
			indiv_net_value = EXCLUDED.indiv_net_value
	`

	for _, flow := range flows {
		batch.Queue(query,
			flow.StockCode, flow.TradeDate,
			flow.ForeignNetQty, flow.ForeignNetValue,
			flow.InstNetQty, flow.InstNetValue,
			flow.IndivNetQty, flow.IndivNetValue,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	count := 0
	for range flows {
		_, err := br.Exec()
		if err != nil {
			return count, fmt.Errorf("batch upsert flow: %w", err)
		}
		count++
	}

	return count, nil
}

// GetByDate 특정 날짜 수급 조회
func (r *FlowRepository) GetByDate(ctx context.Context, stockCode string, date time.Time) (*fetcher.InvestorFlow, error) {
	query := `
		SELECT stock_code, trade_date, foreign_net_qty, foreign_net_value, inst_net_qty, inst_net_value, indiv_net_qty, indiv_net_value, created_at
		FROM data.investor_flow
		WHERE stock_code = $1 AND trade_date = $2
	`

	var flow fetcher.InvestorFlow
	err := r.pool.QueryRow(ctx, query, stockCode, date).Scan(
		&flow.StockCode, &flow.TradeDate,
		&flow.ForeignNetQty, &flow.ForeignNetValue,
		&flow.InstNetQty, &flow.InstNetValue,
		&flow.IndivNetQty, &flow.IndivNetValue,
		&flow.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fetcher.ErrFlowNotFound
		}
		return nil, fmt.Errorf("get flow: %w", err)
	}

	return &flow, nil
}

// GetRange 기간별 수급 조회
func (r *FlowRepository) GetRange(ctx context.Context, stockCode string, from, to time.Time) ([]*fetcher.InvestorFlow, error) {
	query := `
		SELECT stock_code, trade_date, foreign_net_qty, foreign_net_value, inst_net_qty, inst_net_value, indiv_net_qty, indiv_net_value, created_at
		FROM data.investor_flow
		WHERE stock_code = $1 AND trade_date >= $2 AND trade_date <= $3
		ORDER BY trade_date DESC
	`

	rows, err := r.pool.Query(ctx, query, stockCode, from, to)
	if err != nil {
		return nil, fmt.Errorf("query flows: %w", err)
	}
	defer rows.Close()

	var flows []*fetcher.InvestorFlow
	for rows.Next() {
		var flow fetcher.InvestorFlow
		if err := rows.Scan(
			&flow.StockCode, &flow.TradeDate,
			&flow.ForeignNetQty, &flow.ForeignNetValue,
			&flow.InstNetQty, &flow.InstNetValue,
			&flow.IndivNetQty, &flow.IndivNetValue,
			&flow.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan flow: %w", err)
		}
		flows = append(flows, &flow)
	}

	return flows, rows.Err()
}

// GetLatest 최신 수급 조회
func (r *FlowRepository) GetLatest(ctx context.Context, stockCode string) (*fetcher.InvestorFlow, error) {
	query := `
		SELECT stock_code, trade_date, foreign_net_qty, foreign_net_value, inst_net_qty, inst_net_value, indiv_net_qty, indiv_net_value, created_at
		FROM data.investor_flow
		WHERE stock_code = $1
		ORDER BY trade_date DESC
		LIMIT 1
	`

	var flow fetcher.InvestorFlow
	err := r.pool.QueryRow(ctx, query, stockCode).Scan(
		&flow.StockCode, &flow.TradeDate,
		&flow.ForeignNetQty, &flow.ForeignNetValue,
		&flow.InstNetQty, &flow.InstNetValue,
		&flow.IndivNetQty, &flow.IndivNetValue,
		&flow.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fetcher.ErrFlowNotFound
		}
		return nil, fmt.Errorf("get latest flow: %w", err)
	}

	return &flow, nil
}
