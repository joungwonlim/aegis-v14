package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/audit"
)

// Repository audit 데이터 리포지토리
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository 새 리포지토리 생성
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// =============================================================================
// Snapshot Repository
// =============================================================================

// SaveSnapshot 스냅샷 저장
func (r *Repository) SaveSnapshot(ctx context.Context, snapshot *audit.DailySnapshot) error {
	positionsJSON, err := json.Marshal(snapshot.Positions)
	if err != nil {
		return fmt.Errorf("failed to marshal positions: %w", err)
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
		snapshot.Date, snapshot.TotalValue, snapshot.Cash, positionsJSON,
		snapshot.DailyReturn, snapshot.CumReturn,
	)
	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	return nil
}

// GetSnapshot 특정 날짜 스냅샷 조회
func (r *Repository) GetSnapshot(ctx context.Context, date time.Time) (*audit.DailySnapshot, error) {
	query := `
		SELECT date, total_value, cash, positions, daily_return, cum_return
		FROM audit.daily_snapshots
		WHERE date = $1
	`

	var snapshot audit.DailySnapshot
	var positionsJSON []byte

	err := r.pool.QueryRow(ctx, query, date).Scan(
		&snapshot.Date, &snapshot.TotalValue, &snapshot.Cash, &positionsJSON,
		&snapshot.DailyReturn, &snapshot.CumReturn,
	)
	if err == pgx.ErrNoRows {
		return nil, audit.ErrSnapshotNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	if err := json.Unmarshal(positionsJSON, &snapshot.Positions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal positions: %w", err)
	}

	return &snapshot, nil
}

// GetPreviousSnapshot 이전 스냅샷 조회
func (r *Repository) GetPreviousSnapshot(ctx context.Context, date time.Time) (*audit.DailySnapshot, error) {
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
		&snapshot.Date, &snapshot.TotalValue, &snapshot.Cash, &positionsJSON,
		&snapshot.DailyReturn, &snapshot.CumReturn,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get previous snapshot: %w", err)
	}

	if err := json.Unmarshal(positionsJSON, &snapshot.Positions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal positions: %w", err)
	}

	return &snapshot, nil
}

// GetSnapshotHistory 스냅샷 히스토리 조회
func (r *Repository) GetSnapshotHistory(ctx context.Context, startDate, endDate time.Time) ([]audit.DailySnapshot, error) {
	query := `
		SELECT date, total_value, cash, positions, daily_return, cum_return
		FROM audit.daily_snapshots
		WHERE date BETWEEN $1 AND $2
		ORDER BY date ASC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}
	defer rows.Close()

	snapshots := make([]audit.DailySnapshot, 0)

	for rows.Next() {
		var snapshot audit.DailySnapshot
		var positionsJSON []byte

		err := rows.Scan(
			&snapshot.Date, &snapshot.TotalValue, &snapshot.Cash, &positionsJSON,
			&snapshot.DailyReturn, &snapshot.CumReturn,
		)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(positionsJSON, &snapshot.Positions); err != nil {
			continue
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// GetDailyReturns 일별 수익률 조회
func (r *Repository) GetDailyReturns(ctx context.Context, startDate, endDate time.Time) ([]float64, error) {
	query := `
		SELECT daily_return
		FROM audit.daily_snapshots
		WHERE date BETWEEN $1 AND $2
		ORDER BY date ASC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily returns: %w", err)
	}
	defer rows.Close()

	returns := make([]float64, 0)

	for rows.Next() {
		var ret float64
		if err := rows.Scan(&ret); err != nil {
			continue
		}
		returns = append(returns, ret)
	}

	return returns, nil
}

// =============================================================================
// PnL Repository
// =============================================================================

// SaveDailyPnL 일별 손익 저장
func (r *Repository) SaveDailyPnL(ctx context.Context, pnl *audit.DailyPnL) error {
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
		pnl.Date, pnl.RealizedPnL, pnl.UnrealizedPnL, pnl.TotalPnL,
		pnl.DailyReturn, pnl.CumulativeReturn, pnl.PortfolioValue, pnl.CashBalance,
	)
	if err != nil {
		return fmt.Errorf("failed to save daily pnl: %w", err)
	}

	return nil
}

// GetDailyPnL 특정 날짜 손익 조회
func (r *Repository) GetDailyPnL(ctx context.Context, date time.Time) (*audit.DailyPnL, error) {
	query := `
		SELECT pnl_date, realized_pnl, unrealized_pnl, total_pnl,
		       daily_return, cumulative_return, portfolio_value, cash_balance
		FROM audit.daily_pnl
		WHERE pnl_date = $1
	`

	var pnl audit.DailyPnL
	err := r.pool.QueryRow(ctx, query, date).Scan(
		&pnl.Date, &pnl.RealizedPnL, &pnl.UnrealizedPnL, &pnl.TotalPnL,
		&pnl.DailyReturn, &pnl.CumulativeReturn, &pnl.PortfolioValue, &pnl.CashBalance,
	)
	if err == pgx.ErrNoRows {
		return nil, audit.ErrReportNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get daily pnl: %w", err)
	}

	return &pnl, nil
}

// GetPnLHistory 손익 히스토리 조회
func (r *Repository) GetPnLHistory(ctx context.Context, startDate, endDate time.Time) ([]audit.DailyPnL, error) {
	query := `
		SELECT pnl_date, realized_pnl, unrealized_pnl, total_pnl,
		       daily_return, cumulative_return, portfolio_value, cash_balance
		FROM audit.daily_pnl
		WHERE pnl_date BETWEEN $1 AND $2
		ORDER BY pnl_date ASC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query pnl history: %w", err)
	}
	defer rows.Close()

	pnls := make([]audit.DailyPnL, 0)

	for rows.Next() {
		var pnl audit.DailyPnL
		err := rows.Scan(
			&pnl.Date, &pnl.RealizedPnL, &pnl.UnrealizedPnL, &pnl.TotalPnL,
			&pnl.DailyReturn, &pnl.CumulativeReturn, &pnl.PortfolioValue, &pnl.CashBalance,
		)
		if err != nil {
			continue
		}
		pnls = append(pnls, pnl)
	}

	return pnls, nil
}

// =============================================================================
// Report Repository
// =============================================================================

// SavePerformanceReport 성과 리포트 저장
func (r *Repository) SavePerformanceReport(ctx context.Context, report *audit.PerformanceReport) error {
	query := `
		INSERT INTO audit.performance_reports (
			report_date, period_start, period_end, total_return, benchmark_return,
			alpha, beta, sharpe_ratio, sortino_ratio, volatility, max_drawdown,
			win_rate, avg_win, avg_loss, profit_factor, total_trades
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (report_date) DO UPDATE SET
			period_start = EXCLUDED.period_start,
			period_end = EXCLUDED.period_end,
			total_return = EXCLUDED.total_return,
			benchmark_return = EXCLUDED.benchmark_return,
			alpha = EXCLUDED.alpha,
			beta = EXCLUDED.beta,
			sharpe_ratio = EXCLUDED.sharpe_ratio,
			sortino_ratio = EXCLUDED.sortino_ratio,
			volatility = EXCLUDED.volatility,
			max_drawdown = EXCLUDED.max_drawdown,
			win_rate = EXCLUDED.win_rate,
			avg_win = EXCLUDED.avg_win,
			avg_loss = EXCLUDED.avg_loss,
			profit_factor = EXCLUDED.profit_factor,
			total_trades = EXCLUDED.total_trades
	`

	_, err := r.pool.Exec(ctx, query,
		time.Now().Truncate(24*time.Hour), report.StartDate, report.EndDate,
		report.TotalReturn, report.Benchmark, report.Alpha, report.Beta,
		report.Sharpe, report.Sortino, report.Volatility, report.MaxDrawdown,
		report.WinRate, report.AvgWin, report.AvgLoss, report.ProfitFactor, report.TotalTrades,
	)
	if err != nil {
		return fmt.Errorf("failed to save performance report: %w", err)
	}

	return nil
}

// GetPerformanceReport 성과 리포트 조회
func (r *Repository) GetPerformanceReport(ctx context.Context, period audit.Period) (*audit.PerformanceReport, error) {
	query := `
		SELECT period_start, period_end, total_return, benchmark_return,
		       alpha, beta, sharpe_ratio, sortino_ratio, volatility, max_drawdown,
		       win_rate, avg_win, avg_loss, profit_factor, total_trades
		FROM audit.performance_reports
		ORDER BY report_date DESC
		LIMIT 1
	`

	var report audit.PerformanceReport
	report.Period = period

	err := r.pool.QueryRow(ctx, query).Scan(
		&report.StartDate, &report.EndDate, &report.TotalReturn, &report.Benchmark,
		&report.Alpha, &report.Beta, &report.Sharpe, &report.Sortino,
		&report.Volatility, &report.MaxDrawdown,
		&report.WinRate, &report.AvgWin, &report.AvgLoss, &report.ProfitFactor, &report.TotalTrades,
	)
	if err == pgx.ErrNoRows {
		return nil, audit.ErrReportNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get performance report: %w", err)
	}

	return &report, nil
}

// GetLatestReport 최신 리포트 조회
func (r *Repository) GetLatestReport(ctx context.Context) (*audit.PerformanceReport, error) {
	return r.GetPerformanceReport(ctx, audit.Period1M)
}

// SaveAttribution 귀속 분석 저장
func (r *Repository) SaveAttribution(ctx context.Context, analysis *audit.AttributionAnalysis) error {
	factorsJSON, err := json.Marshal(analysis.Factors)
	if err != nil {
		return fmt.Errorf("failed to marshal factors: %w", err)
	}

	sectorsJSON, err := json.Marshal(analysis.Sectors)
	if err != nil {
		return fmt.Errorf("failed to marshal sectors: %w", err)
	}

	stocksJSON, err := json.Marshal(analysis.Stocks)
	if err != nil {
		return fmt.Errorf("failed to marshal stocks: %w", err)
	}

	query := `
		INSERT INTO audit.attribution_analysis (
			analysis_date, period_start, period_end, total_return,
			factor_contrib, sector_contrib, stock_contrib
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
		analysis.AnalysisDate, analysis.PeriodStart, analysis.PeriodEnd,
		analysis.TotalReturn, factorsJSON, sectorsJSON, stocksJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to save attribution: %w", err)
	}

	return nil
}

// GetAttribution 귀속 분석 조회
func (r *Repository) GetAttribution(ctx context.Context, period audit.Period) (*audit.AttributionAnalysis, error) {
	query := `
		SELECT analysis_date, period_start, period_end, total_return,
		       factor_contrib, sector_contrib, stock_contrib
		FROM audit.attribution_analysis
		ORDER BY analysis_date DESC
		LIMIT 1
	`

	var analysis audit.AttributionAnalysis
	var factorsJSON, sectorsJSON, stocksJSON []byte

	err := r.pool.QueryRow(ctx, query).Scan(
		&analysis.AnalysisDate, &analysis.PeriodStart, &analysis.PeriodEnd,
		&analysis.TotalReturn, &factorsJSON, &sectorsJSON, &stocksJSON,
	)
	if err == pgx.ErrNoRows {
		return nil, audit.ErrReportNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attribution: %w", err)
	}

	if err := json.Unmarshal(factorsJSON, &analysis.Factors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal factors: %w", err)
	}
	if err := json.Unmarshal(sectorsJSON, &analysis.Sectors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sectors: %w", err)
	}
	if err := json.Unmarshal(stocksJSON, &analysis.Stocks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stocks: %w", err)
	}

	return &analysis, nil
}

// =============================================================================
// Trade Repository
// =============================================================================

// GetTrades 거래 내역 조회
func (r *Repository) GetTrades(ctx context.Context, startDate, endDate time.Time) ([]audit.Trade, error) {
	query := `
		SELECT symbol, side, quantity, price, pnl, pnl_percent,
		       entry_date, exit_date, hold_days
		FROM audit.trades
		WHERE exit_date BETWEEN $1 AND $2
		ORDER BY exit_date ASC
	`

	rows, err := r.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	trades := make([]audit.Trade, 0)

	for rows.Next() {
		var t audit.Trade
		err := rows.Scan(
			&t.Symbol, &t.Side, &t.Quantity, &t.Price, &t.PnL, &t.PnLPercent,
			&t.EntryDate, &t.ExitDate, &t.HoldDays,
		)
		if err != nil {
			continue
		}
		trades = append(trades, t)
	}

	return trades, nil
}

// GetTradesBySymbol 종목별 거래 내역 조회
func (r *Repository) GetTradesBySymbol(ctx context.Context, symbol string, startDate, endDate time.Time) ([]audit.Trade, error) {
	query := `
		SELECT symbol, side, quantity, price, pnl, pnl_percent,
		       entry_date, exit_date, hold_days
		FROM audit.trades
		WHERE symbol = $1 AND exit_date BETWEEN $2 AND $3
		ORDER BY exit_date ASC
	`

	rows, err := r.pool.Query(ctx, query, symbol, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades by symbol: %w", err)
	}
	defer rows.Close()

	trades := make([]audit.Trade, 0)

	for rows.Next() {
		var t audit.Trade
		err := rows.Scan(
			&t.Symbol, &t.Side, &t.Quantity, &t.Price, &t.PnL, &t.PnLPercent,
			&t.EntryDate, &t.ExitDate, &t.HoldDays,
		)
		if err != nil {
			continue
		}
		trades = append(trades, t)
	}

	return trades, nil
}

// =============================================================================
// Benchmark Repository
// =============================================================================

// SaveBenchmark 벤치마크 데이터 저장
func (r *Repository) SaveBenchmark(ctx context.Context, data *audit.BenchmarkData) error {
	query := `
		INSERT INTO audit.benchmark_data (
			benchmark_date, benchmark_code, close_price, daily_return
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (benchmark_date, benchmark_code) DO UPDATE SET
			close_price = EXCLUDED.close_price,
			daily_return = EXCLUDED.daily_return
	`

	_, err := r.pool.Exec(ctx, query,
		data.Date, data.Code, data.ClosePrice, data.DailyReturn,
	)
	if err != nil {
		return fmt.Errorf("failed to save benchmark: %w", err)
	}

	return nil
}

// GetBenchmark 벤치마크 데이터 조회
func (r *Repository) GetBenchmark(ctx context.Context, code string, startDate, endDate time.Time) ([]audit.BenchmarkData, error) {
	query := `
		SELECT benchmark_date, benchmark_code, close_price, daily_return
		FROM audit.benchmark_data
		WHERE benchmark_code = $1 AND benchmark_date BETWEEN $2 AND $3
		ORDER BY benchmark_date ASC
	`

	rows, err := r.pool.Query(ctx, query, code, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query benchmark: %w", err)
	}
	defer rows.Close()

	data := make([]audit.BenchmarkData, 0)

	for rows.Next() {
		var d audit.BenchmarkData
		err := rows.Scan(&d.Date, &d.Code, &d.ClosePrice, &d.DailyReturn)
		if err != nil {
			continue
		}
		data = append(data, d)
	}

	return data, nil
}

// GetBenchmarkReturns 벤치마크 수익률 조회
func (r *Repository) GetBenchmarkReturns(ctx context.Context, code string, startDate, endDate time.Time) ([]float64, error) {
	query := `
		SELECT daily_return
		FROM audit.benchmark_data
		WHERE benchmark_code = $1 AND benchmark_date BETWEEN $2 AND $3
		ORDER BY benchmark_date ASC
	`

	rows, err := r.pool.Query(ctx, query, code, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query benchmark returns: %w", err)
	}
	defer rows.Close()

	returns := make([]float64, 0)

	for rows.Next() {
		var ret float64
		if err := rows.Scan(&ret); err != nil {
			continue
		}
		returns = append(returns, ret)
	}

	return returns, nil
}
