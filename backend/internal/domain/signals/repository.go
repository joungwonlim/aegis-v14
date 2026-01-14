package signals

import (
	"context"
	"time"
)

// SignalRepository 신호 저장소
type SignalRepository interface {
	// 스냅샷 저장
	SaveSnapshot(ctx context.Context, snapshot *SignalSnapshot) error

	// 최신 스냅샷 조회
	GetLatestSnapshot(ctx context.Context) (*SignalSnapshot, error)

	// 특정 스냅샷 조회
	GetSnapshotByID(ctx context.Context, snapshotID string) (*SignalSnapshot, error)

	// 스냅샷 목록 (시간 범위)
	ListSnapshots(ctx context.Context, from, to time.Time) ([]*SignalSnapshot, error)

	// 특정 종목의 신호 조회
	GetSignalBySymbol(ctx context.Context, snapshotID, symbol string) (*Signal, error)
}

// FactorRepository 팩터 데이터 저장소
type FactorRepository interface {
	// 모멘텀 데이터 조회
	GetMomentumFactors(ctx context.Context, symbol string) (*MomentumFactors, error)

	// 품질 데이터 조회
	GetQualityFactors(ctx context.Context, symbol string) (*QualityFactors, error)

	// 가치 데이터 조회
	GetValueFactors(ctx context.Context, symbol string) (*ValueFactors, error)

	// 기술적 지표 조회
	GetTechnicalFactors(ctx context.Context, symbol string) (*TechnicalFactors, error)
}
