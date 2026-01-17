package audit

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/audit"
)

// =============================================================================
// Audit Service
// =============================================================================

// Service 성과 분석 서비스
type Service struct {
	repo audit.Repository
}

// NewService 새 서비스 생성
func NewService(repo audit.Repository) *Service {
	return &Service{repo: repo}
}

// =============================================================================
// Performance Report
// =============================================================================

// GeneratePerformanceReport 성과 리포트 생성
func (s *Service) GeneratePerformanceReport(ctx context.Context, period audit.Period) (*audit.PerformanceReport, error) {
	if !period.IsValid() {
		return nil, audit.ErrInvalidPeriod
	}

	startDate, endDate := s.calculateDateRange(period)

	// 일별 수익률 조회
	dailyReturns, err := s.repo.GetDailyReturns(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if len(dailyReturns) < 5 {
		return nil, audit.ErrInsufficientData
	}

	// 수익률 계산
	totalReturn := CalculateTotalReturn(dailyReturns)
	annualReturn := CalculateAnnualizedReturn(totalReturn, len(dailyReturns))

	// 리스크 지표 계산
	volatility := CalculateVolatility(dailyReturns)
	sharpe := CalculateSharpe(annualReturn, volatility)
	sortino := CalculateSortino(dailyReturns)
	maxDrawdown := CalculateMaxDrawdown(dailyReturns)

	// 거래 지표 계산
	trades, err := s.repo.GetTrades(ctx, startDate, endDate)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get trades")
		trades = []audit.Trade{}
	}

	tradingMetrics := CalculateTradingMetrics(trades)

	// 벤치마크 비교 (KOSPI)
	benchmarkReturns, err := s.repo.GetBenchmarkReturns(ctx, "KOSPI", startDate, endDate)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get benchmark returns")
		benchmarkReturns = nil
	}

	var benchmark, alpha, beta float64
	if len(benchmarkReturns) > 0 {
		benchmark = CalculateTotalReturn(benchmarkReturns)
		alpha = CalculateAlpha(totalReturn, benchmark)
		beta = CalculateBeta(dailyReturns, benchmarkReturns)
	}

	report := &audit.PerformanceReport{
		Period:       period,
		StartDate:    startDate,
		EndDate:      endDate,
		TotalReturn:  totalReturn,
		AnnualReturn: annualReturn,
		Volatility:   volatility,
		Sharpe:       sharpe,
		Sortino:      sortino,
		MaxDrawdown:  maxDrawdown,
		WinRate:      tradingMetrics.WinRate,
		AvgWin:       tradingMetrics.AvgWin,
		AvgLoss:      tradingMetrics.AvgLoss,
		ProfitFactor: tradingMetrics.ProfitFactor,
		TotalTrades:  tradingMetrics.TotalTrades,
		Benchmark:    benchmark,
		Alpha:        alpha,
		Beta:         beta,
	}

	// 리포트 저장
	if err := s.repo.SavePerformanceReport(ctx, report); err != nil {
		log.Warn().Err(err).Msg("Failed to save performance report")
	}

	return report, nil
}

// GetPerformanceReport 성과 리포트 조회
func (s *Service) GetPerformanceReport(ctx context.Context, period audit.Period) (*audit.PerformanceReport, error) {
	if !period.IsValid() {
		return nil, audit.ErrInvalidPeriod
	}

	report, err := s.repo.GetPerformanceReport(ctx, period)
	if err != nil {
		// 캐시된 리포트 없으면 생성
		return s.GeneratePerformanceReport(ctx, period)
	}

	return report, nil
}

// =============================================================================
// Daily PnL
// =============================================================================

// GetDailyPnLHistory 일별 손익 히스토리 조회
func (s *Service) GetDailyPnLHistory(ctx context.Context, startDate, endDate time.Time) ([]audit.DailyPnL, error) {
	return s.repo.GetPnLHistory(ctx, startDate, endDate)
}

// RecordDailyPnL 일별 손익 기록
func (s *Service) RecordDailyPnL(ctx context.Context, pnl *audit.DailyPnL) error {
	return s.repo.SaveDailyPnL(ctx, pnl)
}

// =============================================================================
// Attribution Analysis
// =============================================================================

// GenerateAttributionAnalysis 귀속 분석 생성
func (s *Service) GenerateAttributionAnalysis(ctx context.Context, period audit.Period) (*audit.AttributionAnalysis, error) {
	if !period.IsValid() {
		return nil, audit.ErrInvalidPeriod
	}

	startDate, endDate := s.calculateDateRange(period)

	// 일별 수익률 조회
	dailyReturns, err := s.repo.GetDailyReturns(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	totalReturn := CalculateTotalReturn(dailyReturns)

	// 팩터 기여도 계산 (placeholder - 실제로는 signals 모듈과 연동)
	factors := []audit.FactorAttribution{
		{Factor: "momentum", Contribution: 0, Exposure: 0, ReturnPct: 0},
		{Factor: "technical", Contribution: 0, Exposure: 0, ReturnPct: 0},
		{Factor: "value", Contribution: 0, Exposure: 0, ReturnPct: 0},
		{Factor: "quality", Contribution: 0, Exposure: 0, ReturnPct: 0},
		{Factor: "flow", Contribution: 0, Exposure: 0, ReturnPct: 0},
		{Factor: "event", Contribution: 0, Exposure: 0, ReturnPct: 0},
	}

	analysis := &audit.AttributionAnalysis{
		AnalysisDate: time.Now(),
		PeriodStart:  startDate,
		PeriodEnd:    endDate,
		TotalReturn:  totalReturn,
		Factors:      factors,
		Sectors:      make(map[string]float64),
		Stocks:       make(map[string]float64),
	}

	// 분석 저장
	if err := s.repo.SaveAttribution(ctx, analysis); err != nil {
		log.Warn().Err(err).Msg("Failed to save attribution analysis")
	}

	return analysis, nil
}

// GetAttributionAnalysis 귀속 분석 조회
func (s *Service) GetAttributionAnalysis(ctx context.Context, period audit.Period) (*audit.AttributionAnalysis, error) {
	if !period.IsValid() {
		return nil, audit.ErrInvalidPeriod
	}

	analysis, err := s.repo.GetAttribution(ctx, period)
	if err != nil {
		return s.GenerateAttributionAnalysis(ctx, period)
	}

	return analysis, nil
}

// =============================================================================
// Risk Metrics
// =============================================================================

// CalculateRiskMetrics 리스크 지표 계산
func (s *Service) CalculateRiskMetrics(ctx context.Context, period audit.Period) (*audit.RiskMetrics, error) {
	if !period.IsValid() {
		return nil, audit.ErrInvalidPeriod
	}

	startDate, endDate := s.calculateDateRange(period)

	dailyReturns, err := s.repo.GetDailyReturns(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if len(dailyReturns) < 5 {
		return nil, audit.ErrInsufficientData
	}

	return &audit.RiskMetrics{
		VaR95:       CalculateVaR(dailyReturns, 0.95),
		VaR99:       CalculateVaR(dailyReturns, 0.99),
		CVaR95:      CalculateCVaR(dailyReturns, 0.95),
		CVaR99:      CalculateCVaR(dailyReturns, 0.99),
		MaxDrawdown: CalculateMaxDrawdown(dailyReturns),
		Volatility:  CalculateVolatility(dailyReturns),
	}, nil
}

// =============================================================================
// Snapshot
// =============================================================================

// RecordDailySnapshot 일별 스냅샷 기록
func (s *Service) RecordDailySnapshot(ctx context.Context, snapshot *audit.DailySnapshot) error {
	// 이전 스냅샷 조회하여 수익률 계산
	prevSnapshot, err := s.repo.GetPreviousSnapshot(ctx, snapshot.Date)
	if err == nil && prevSnapshot != nil && prevSnapshot.TotalValue > 0 {
		snapshot.DailyReturn = float64(snapshot.TotalValue-prevSnapshot.TotalValue) / float64(prevSnapshot.TotalValue)
		snapshot.CumReturn = (1 + prevSnapshot.CumReturn) * (1 + snapshot.DailyReturn) - 1
	} else {
		snapshot.DailyReturn = 0
		snapshot.CumReturn = 0
	}

	return s.repo.SaveSnapshot(ctx, snapshot)
}

// GetSnapshotHistory 스냅샷 히스토리 조회
func (s *Service) GetSnapshotHistory(ctx context.Context, startDate, endDate time.Time) ([]audit.DailySnapshot, error) {
	return s.repo.GetSnapshotHistory(ctx, startDate, endDate)
}

// =============================================================================
// Helper Methods
// =============================================================================

// calculateDateRange 기간별 날짜 범위 계산
func (s *Service) calculateDateRange(period audit.Period) (time.Time, time.Time) {
	endDate := time.Now()
	var startDate time.Time

	switch period {
	case audit.Period1M:
		startDate = endDate.AddDate(0, -1, 0)
	case audit.Period3M:
		startDate = endDate.AddDate(0, -3, 0)
	case audit.Period6M:
		startDate = endDate.AddDate(0, -6, 0)
	case audit.Period1Y:
		startDate = endDate.AddDate(-1, 0, 0)
	case audit.PeriodYTD:
		startDate = time.Date(endDate.Year(), 1, 1, 0, 0, 0, 0, endDate.Location())
	default:
		startDate = endDate.AddDate(0, -1, 0)
	}

	return startDate, endDate
}
