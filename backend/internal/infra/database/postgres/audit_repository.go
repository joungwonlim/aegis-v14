package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/audit"
)

// =============================================================================
// Audit Repository
// =============================================================================

// AuditRepository PostgreSQL 기반 Audit 리포지토리
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository 새 Audit 리포지토리 생성
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// =============================================================================
// Snapshot Repository
// =============================================================================

// SaveSnapshot 일별 스냅샷 저장
func (r *AuditRepository) SaveSnapshot(ctx context.Context, snapshot *audit.DailySnapshot) error {
	positionsJSON, err := json.Marshal(snapshot.Positions)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO audit.daily_snapshots (
			date, total_value, cash, positions, daily_return, cum_return
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (date) DO UPDATE SET
			total_value = EXCLUDED.total_value,
			cash = EXCLUDED.cash,
			positions = EXCLUDED.positions,
			daily_return = EXCLUDED.daily_return,
			cum_return = EXCLUDED.cum_return
	`

	_, err = r.pool.Exec(ctx, query,
		snapshot.Date,
		snapshot.TotalValue,
		snapshot.Cash,
		positionsJSON,
		snapshot.DailyReturn,
		snapshot.CumReturn,
	)

	return err
}

// GetSnapshot 특정 날짜의 스냅샷 조회
func (r *AuditRepository) GetSnapshot(ctx context.Context, date time.Time) (*audit.DailySnapshot, error) {
	query := `
		SELECT date, total_value, cash, positions, daily_return, cum_return
		FROM audit.daily_snapshots
		WHERE date = $1
	`

	var snapshot audit.DailySnapshot
	var positionsJSON []byte

	err := r.pool.QueryRow(ctx, query, date).Scan(
		&snapshot.Date,
		&snapshot.TotalValue,
		&snapshot.Cash,
		&positionsJSON,
		&snapshot.DailyReturn,
		&snapshot.CumReturn,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(positionsJSON, &snapshot.Positions); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// GetPreviousSnapshot 이전 스냅샷 조회
func (r *AuditRepository) GetPreviousSnapshot(ctx context.Context, date time.Time) (*audit.DailySnapshot, error) {
	query := `
		SELECT date, total_value, cash, positions, daily_return, cum_return
		FROM audit.daily_snapshots
		WHERE date < $1
		ORDER BY date DESC
		LIMIT 1
	`

	var snapshot audit.DailySnapshot
	var positionsJSON []byte

	err := r.pool.QueryRow(ctx, query, date).Scan(
		&snapshot.Date,
		&snapshot.TotalValue,
		&snapshot.Cash,
		&positionsJSON,
		&snapshot.DailyReturn,
		&snapshot.CumReturn,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(positionsJSON, &snapshot.Positions); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// GetSnapshotHistory 기간별 스냅샷 이력 조회
func (r *AuditRepository) GetSnapshotHistory(ctx context.Context, startDate, endDate time.Time) ([]audit.DailySnapshot, error) {
	query := `
		SELECT date, total_value, cash, positions, daily_return, cum_return
		FROM audit.daily_snapshots
		WHERE date BETWEEN $1 AND $2
		ORDER BY date ASC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []audit.DailySnapshot
	for rows.Next() {
		var snapshot audit.DailySnapshot
		var positionsJSON []byte

		if err := rows.Scan(
			&snapshot.Date,
			&snapshot.TotalValue,
			&snapshot.Cash,
			&positionsJSON,
			&snapshot.DailyReturn,
			&snapshot.CumReturn,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(positionsJSON, &snapshot.Positions); err != nil {
			return nil, err
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, rows.Err()
}

// GetDailyReturns 일별 수익률 조회
func (r *AuditRepository) GetDailyReturns(ctx context.Context, startDate, endDate time.Time) ([]float64, error) {
	query := `
		SELECT daily_return
		FROM audit.daily_snapshots
		WHERE date BETWEEN $1 AND $2
		AND daily_return IS NOT NULL
		ORDER BY date ASC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var returns []float64
	for rows.Next() {
		var ret float64
		if err := rows.Scan(&ret); err != nil {
			return nil, err
		}
		returns = append(returns, ret)
	}

	return returns, rows.Err()
}

// =============================================================================
// PnL Repository
// =============================================================================

// SaveDailyPnL 일별 손익 저장
func (r *AuditRepository) SaveDailyPnL(ctx context.Context, pnl *audit.DailyPnL) error {
	query := `
		INSERT INTO audit.daily_pnl (
			pnl_date, realized_pnl, unrealized_pnl, total_pnl,
			daily_return, cumulative_return, portfolio_value, cash_balance
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (pnl_date) DO UPDATE SET
			realized_pnl = EXCLUDED.realized_pnl,
			unrealized_pnl = EXCLUDED.unrealized_pnl,
			total_pnl = EXCLUDED.total_pnl,
			daily_return = EXCLUDED.daily_return,
			cumulative_return = EXCLUDED.cumulative_return,
			portfolio_value = EXCLUDED.portfolio_value,
			cash_balance = EXCLUDED.cash_balance
	`

	_, err := r.pool.Exec(ctx, query,
		pnl.Date,
		pnl.RealizedPnL,
		pnl.UnrealizedPnL,
		pnl.TotalPnL,
		pnl.DailyReturn,
		pnl.CumulativeReturn,
		pnl.PortfolioValue,
		pnl.CashBalance,
	)

	return err
}

// GetDailyPnL 특정 날짜의 손익 조회
func (r *AuditRepository) GetDailyPnL(ctx context.Context, date time.Time) (*audit.DailyPnL, error) {
	query := `
		SELECT pnl_date, realized_pnl, unrealized_pnl, total_pnl,
			   daily_return, cumulative_return, portfolio_value, cash_balance
		FROM audit.daily_pnl
		WHERE pnl_date = $1
	`

	var pnl audit.DailyPnL
	err := r.pool.QueryRow(ctx, query, date).Scan(
		&pnl.Date,
		&pnl.RealizedPnL,
		&pnl.UnrealizedPnL,
		&pnl.TotalPnL,
		&pnl.DailyReturn,
		&pnl.CumulativeReturn,
		&pnl.PortfolioValue,
		&pnl.CashBalance,
	)

	if err != nil {
		return nil, err
	}

	return &pnl, nil
}

// GetPnLHistory 기간별 손익 이력 조회
func (r *AuditRepository) GetPnLHistory(ctx context.Context, startDate, endDate time.Time) ([]audit.DailyPnL, error) {
	query := `
		SELECT pnl_date, realized_pnl, unrealized_pnl, total_pnl,
			   daily_return, cumulative_return, portfolio_value, cash_balance
		FROM audit.daily_pnl
		WHERE pnl_date BETWEEN $1 AND $2
		ORDER BY pnl_date ASC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pnls []audit.DailyPnL
	for rows.Next() {
		var pnl audit.DailyPnL
		if err := rows.Scan(
			&pnl.Date,
			&pnl.RealizedPnL,
			&pnl.UnrealizedPnL,
			&pnl.TotalPnL,
			&pnl.DailyReturn,
			&pnl.CumulativeReturn,
			&pnl.PortfolioValue,
			&pnl.CashBalance,
		); err != nil {
			return nil, err
		}
		pnls = append(pnls, pnl)
	}

	return pnls, rows.Err()
}

// =============================================================================
// Report Repository
// =============================================================================

// SavePerformanceReport 성과 리포트 저장
func (r *AuditRepository) SavePerformanceReport(ctx context.Context, report *audit.PerformanceReport) error {
	query := `
		INSERT INTO audit.performance_reports (
			report_date, period_start, period_end, period_code,
			total_return, annual_return,
			benchmark_return, alpha, beta,
			volatility, sharpe_ratio, sortino_ratio, max_drawdown,
			win_rate, avg_win, avg_loss, profit_factor, total_trades
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (report_date) DO UPDATE SET
			period_start = EXCLUDED.period_start,
			period_end = EXCLUDED.period_end,
			period_code = EXCLUDED.period_code,
			total_return = EXCLUDED.total_return,
			annual_return = EXCLUDED.annual_return,
			benchmark_return = EXCLUDED.benchmark_return,
			alpha = EXCLUDED.alpha,
			beta = EXCLUDED.beta,
			volatility = EXCLUDED.volatility,
			sharpe_ratio = EXCLUDED.sharpe_ratio,
			sortino_ratio = EXCLUDED.sortino_ratio,
			max_drawdown = EXCLUDED.max_drawdown,
			win_rate = EXCLUDED.win_rate,
			avg_win = EXCLUDED.avg_win,
			avg_loss = EXCLUDED.avg_loss,
			profit_factor = EXCLUDED.profit_factor,
			total_trades = EXCLUDED.total_trades
	`

	_, err := r.pool.Exec(ctx, query,
		time.Now().Truncate(24*time.Hour), // report_date
		report.StartDate,
		report.EndDate,
		report.Period,
		report.TotalReturn,
		report.AnnualReturn,
		report.Benchmark,
		report.Alpha,
		report.Beta,
		report.Volatility,
		report.Sharpe,
		report.Sortino,
		report.MaxDrawdown,
		report.WinRate,
		report.AvgWin,
		report.AvgLoss,
		report.ProfitFactor,
		report.TotalTrades,
	)

	return err
}

// GetPerformanceReport 특정 기간의 성과 리포트 조회
func (r *AuditRepository) GetPerformanceReport(ctx context.Context, period audit.Period) (*audit.PerformanceReport, error) {
	query := `
		SELECT period_start, period_end, period_code,
			   total_return, annual_return,
			   benchmark_return, alpha, beta,
			   volatility, sharpe_ratio, sortino_ratio, max_drawdown,
			   win_rate, avg_win, avg_loss, profit_factor, total_trades
		FROM audit.performance_reports
		WHERE period_code = $1
		ORDER BY report_date DESC
		LIMIT 1
	`

	var report audit.PerformanceReport
	err := r.pool.QueryRow(ctx, query, string(period)).Scan(
		&report.StartDate,
		&report.EndDate,
		&report.Period,
		&report.TotalReturn,
		&report.AnnualReturn,
		&report.Benchmark,
		&report.Alpha,
		&report.Beta,
		&report.Volatility,
		&report.Sharpe,
		&report.Sortino,
		&report.MaxDrawdown,
		&report.WinRate,
		&report.AvgWin,
		&report.AvgLoss,
		&report.ProfitFactor,
		&report.TotalTrades,
	)

	if err != nil {
		return nil, err
	}

	return &report, nil
}

// GetLatestReport 최신 리포트 조회
func (r *AuditRepository) GetLatestReport(ctx context.Context) (*audit.PerformanceReport, error) {
	query := `
		SELECT period_start, period_end, period_code,
			   total_return, annual_return,
			   benchmark_return, alpha, beta,
			   volatility, sharpe_ratio, sortino_ratio, max_drawdown,
			   win_rate, avg_win, avg_loss, profit_factor, total_trades
		FROM audit.performance_reports
		ORDER BY report_date DESC
		LIMIT 1
	`

	var report audit.PerformanceReport
	err := r.pool.QueryRow(ctx, query).Scan(
		&report.StartDate,
		&report.EndDate,
		&report.Period,
		&report.TotalReturn,
		&report.AnnualReturn,
		&report.Benchmark,
		&report.Alpha,
		&report.Beta,
		&report.Volatility,
		&report.Sharpe,
		&report.Sortino,
		&report.MaxDrawdown,
		&report.WinRate,
		&report.AvgWin,
		&report.AvgLoss,
		&report.ProfitFactor,
		&report.TotalTrades,
	)

	if err != nil {
		return nil, err
	}

	return &report, nil
}

// SaveAttribution 귀속 분석 저장
func (r *AuditRepository) SaveAttribution(ctx context.Context, analysis *audit.AttributionAnalysis) error {
	factorJSON, err := json.Marshal(analysis.Factors)
	if err != nil {
		return err
	}

	sectorJSON, err := json.Marshal(analysis.Sectors)
	if err != nil {
		return err
	}

	stockJSON, err := json.Marshal(analysis.Stocks)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO audit.attribution_analysis (
			analysis_date, period_start, period_end,
			total_return, factor_contrib, sector_contrib, stock_contrib
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (analysis_date) DO UPDATE SET
			period_start = EXCLUDED.period_start,
			period_end = EXCLUDED.period_end,
			total_return = EXCLUDED.total_return,
			factor_contrib = EXCLUDED.factor_contrib,
			sector_contrib = EXCLUDED.sector_contrib,
			stock_contrib = EXCLUDED.stock_contrib
	`

	_, err = r.pool.Exec(ctx, query,
		analysis.AnalysisDate,
		analysis.PeriodStart,
		analysis.PeriodEnd,
		analysis.TotalReturn,
		factorJSON,
		sectorJSON,
		stockJSON,
	)

	return err
}

// GetAttribution 귀속 분석 조회
func (r *AuditRepository) GetAttribution(ctx context.Context, period audit.Period) (*audit.AttributionAnalysis, error) {
	// 최신 귀속 분석 조회
	query := `
		SELECT analysis_date, period_start, period_end,
			   total_return, factor_contrib, sector_contrib, stock_contrib
		FROM audit.attribution_analysis
		ORDER BY analysis_date DESC
		LIMIT 1
	`

	var analysis audit.AttributionAnalysis
	var factorJSON, sectorJSON, stockJSON []byte

	err := r.pool.QueryRow(ctx, query).Scan(
		&analysis.AnalysisDate,
		&analysis.PeriodStart,
		&analysis.PeriodEnd,
		&analysis.TotalReturn,
		&factorJSON,
		&sectorJSON,
		&stockJSON,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(factorJSON, &analysis.Factors); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(sectorJSON, &analysis.Sectors); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(stockJSON, &analysis.Stocks); err != nil {
		return nil, err
	}

	return &analysis, nil
}

// =============================================================================
// Trade Repository
// =============================================================================

// GetTrades 거래 내역 조회
func (r *AuditRepository) GetTrades(ctx context.Context, startDate, endDate time.Time) ([]audit.Trade, error) {
	query := `
		SELECT symbol, side, quantity, price, pnl, pnl_percent,
			   entry_date, exit_date, hold_days
		FROM audit.trades
		WHERE exit_date BETWEEN $1 AND $2
		ORDER BY exit_date ASC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []audit.Trade
	for rows.Next() {
		var trade audit.Trade
		if err := rows.Scan(
			&trade.Symbol,
			&trade.Side,
			&trade.Quantity,
			&trade.Price,
			&trade.PnL,
			&trade.PnLPercent,
			&trade.EntryDate,
			&trade.ExitDate,
			&trade.HoldDays,
		); err != nil {
			return nil, err
		}
		trades = append(trades, trade)
	}

	return trades, rows.Err()
}

// GetTradesBySymbol 종목별 거래 내역 조회
func (r *AuditRepository) GetTradesBySymbol(ctx context.Context, symbol string, startDate, endDate time.Time) ([]audit.Trade, error) {
	query := `
		SELECT symbol, side, quantity, price, pnl, pnl_percent,
			   entry_date, exit_date, hold_days
		FROM audit.trades
		WHERE symbol = $1
		  AND exit_date BETWEEN $2 AND $3
		ORDER BY exit_date ASC
	`

	rows, err := r.pool.Query(ctx, query, symbol, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []audit.Trade
	for rows.Next() {
		var trade audit.Trade
		if err := rows.Scan(
			&trade.Symbol,
			&trade.Side,
			&trade.Quantity,
			&trade.Price,
			&trade.PnL,
			&trade.PnLPercent,
			&trade.EntryDate,
			&trade.ExitDate,
			&trade.HoldDays,
		); err != nil {
			return nil, err
		}
		trades = append(trades, trade)
	}

	return trades, rows.Err()
}

// =============================================================================
// Benchmark Repository
// =============================================================================

// SaveBenchmark 벤치마크 데이터 저장
func (r *AuditRepository) SaveBenchmark(ctx context.Context, data *audit.BenchmarkData) error {
	query := `
		INSERT INTO audit.benchmark_data (
			benchmark_date, benchmark_code, close_price, daily_return
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (benchmark_date, benchmark_code) DO UPDATE SET
			close_price = EXCLUDED.close_price,
			daily_return = EXCLUDED.daily_return
	`

	_, err := r.pool.Exec(ctx, query,
		data.Date,
		data.Code,
		data.ClosePrice,
		data.DailyReturn,
	)

	return err
}

// GetBenchmark 벤치마크 데이터 조회
func (r *AuditRepository) GetBenchmark(ctx context.Context, code string, startDate, endDate time.Time) ([]audit.BenchmarkData, error) {
	query := `
		SELECT benchmark_date, benchmark_code, close_price, daily_return
		FROM audit.benchmark_data
		WHERE benchmark_code = $1
		  AND benchmark_date BETWEEN $2 AND $3
		ORDER BY benchmark_date ASC
	`

	rows, err := r.pool.Query(ctx, query, code, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var benchmarks []audit.BenchmarkData
	for rows.Next() {
		var benchmark audit.BenchmarkData
		if err := rows.Scan(
			&benchmark.Date,
			&benchmark.Code,
			&benchmark.ClosePrice,
			&benchmark.DailyReturn,
		); err != nil {
			return nil, err
		}
		benchmarks = append(benchmarks, benchmark)
	}

	return benchmarks, rows.Err()
}

// GetBenchmarkReturns 벤치마크 수익률 조회
func (r *AuditRepository) GetBenchmarkReturns(ctx context.Context, code string, startDate, endDate time.Time) ([]float64, error) {
	query := `
		SELECT daily_return
		FROM audit.benchmark_data
		WHERE benchmark_code = $1
		  AND benchmark_date BETWEEN $2 AND $3
		  AND daily_return IS NOT NULL
		ORDER BY benchmark_date ASC
	`

	rows, err := r.pool.Query(ctx, query, code, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var returns []float64
	for rows.Next() {
		var ret float64
		if err := rows.Scan(&ret); err != nil {
			return nil, err
		}
		returns = append(returns, ret)
	}

	return returns, rows.Err()
}
