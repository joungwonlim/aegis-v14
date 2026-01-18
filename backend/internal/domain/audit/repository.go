package audit

import (
	"context"
	"time"
)

// =============================================================================
// Repository Interfaces
// =============================================================================

// SnapshotRepository 일별 스냅샷 리포지토리 인터페이스
type SnapshotRepository interface {
	// 스냅샷 저장/조회
	SaveSnapshot(ctx context.Context, snapshot *DailySnapshot) error
	GetSnapshot(ctx context.Context, date time.Time) (*DailySnapshot, error)
	GetPreviousSnapshot(ctx context.Context, date time.Time) (*DailySnapshot, error)
	GetSnapshotHistory(ctx context.Context, startDate, endDate time.Time) ([]DailySnapshot, error)

	// 일별 수익률 조회
	GetDailyReturns(ctx context.Context, startDate, endDate time.Time) ([]float64, error)
}

// PnLRepository 손익 리포지토리 인터페이스
type PnLRepository interface {
	// 일별 손익 저장/조회
	SaveDailyPnL(ctx context.Context, pnl *DailyPnL) error
	GetDailyPnL(ctx context.Context, date time.Time) (*DailyPnL, error)
	GetPnLHistory(ctx context.Context, startDate, endDate time.Time) ([]DailyPnL, error)
}

// ReportRepository 리포트 리포지토리 인터페이스
type ReportRepository interface {
	// 성과 리포트 저장/조회
	SavePerformanceReport(ctx context.Context, report *PerformanceReport) error
	GetPerformanceReport(ctx context.Context, period Period) (*PerformanceReport, error)
	GetLatestReport(ctx context.Context) (*PerformanceReport, error)

	// 귀속 분석 저장/조회
	SaveAttribution(ctx context.Context, analysis *AttributionAnalysis) error
	GetAttribution(ctx context.Context, period Period) (*AttributionAnalysis, error)
}

// TradeRepository 거래 리포지토리 인터페이스
type TradeRepository interface {
	// 거래 내역 조회
	GetTrades(ctx context.Context, startDate, endDate time.Time) ([]Trade, error)
	GetTradesBySymbol(ctx context.Context, symbol string, startDate, endDate time.Time) ([]Trade, error)

	// 거래 내역 저장
	SaveTradeHistory(ctx context.Context, trade *Trade) error

	// 마지막 거래 날짜 조회 (증분 동기화용)
	GetLastTradeDate(ctx context.Context) (time.Time, error)

	// 거래 개수 조회
	GetTradeCount(ctx context.Context) (int, error)
}

// BenchmarkRepository 벤치마크 리포지토리 인터페이스
type BenchmarkRepository interface {
	// 벤치마크 데이터 저장/조회
	SaveBenchmark(ctx context.Context, data *BenchmarkData) error
	GetBenchmark(ctx context.Context, code string, startDate, endDate time.Time) ([]BenchmarkData, error)
	GetBenchmarkReturns(ctx context.Context, code string, startDate, endDate time.Time) ([]float64, error)
}

// =============================================================================
// Composite Repository
// =============================================================================

// Repository 통합 리포지토리 인터페이스
type Repository interface {
	SnapshotRepository
	PnLRepository
	ReportRepository
	TradeRepository
	BenchmarkRepository
}
