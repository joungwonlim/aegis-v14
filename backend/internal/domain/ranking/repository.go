package ranking

import (
	"context"
	"time"
)

// RankingRepository 순위 저장소
type RankingRepository interface {
	// 스냅샷 저장
	SaveSnapshot(ctx context.Context, snapshot *RankingSnapshot) error

	// 최신 스냅샷 조회
	GetLatestSnapshot(ctx context.Context) (*RankingSnapshot, error)

	// 특정 스냅샷 조회
	GetSnapshotByID(ctx context.Context, snapshotID string) (*RankingSnapshot, error)

	// 스냅샷 목록 (시간 범위)
	ListSnapshots(ctx context.Context, from, to time.Time) ([]*RankingSnapshot, error)

	// 특정 종목의 순위 조회
	GetRankBySymbol(ctx context.Context, snapshotID, symbol string) (*RankedStock, error)
}

// RiskDataRepository 리스크 데이터 저장소
type RiskDataRepository interface {
	// 리스크 데이터 조회
	GetRiskData(ctx context.Context, symbol string) (*RiskData, error)

	// 배치 리스크 데이터 조회
	GetRiskDataBatch(ctx context.Context, symbols []string) (map[string]*RiskData, error)
}
