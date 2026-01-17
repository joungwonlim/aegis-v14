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

// FactorRepository 팩터 데이터 저장소 (6팩터)
type FactorRepository interface {
	// 모멘텀 데이터 조회
	GetMomentumFactors(ctx context.Context, symbol string) (*MomentumFactors, error)

	// 품질 데이터 조회
	GetQualityFactors(ctx context.Context, symbol string) (*QualityFactors, error)

	// 가치 데이터 조회
	GetValueFactors(ctx context.Context, symbol string) (*ValueFactors, error)

	// 기술적 지표 조회
	GetTechnicalFactors(ctx context.Context, symbol string) (*TechnicalFactors, error)

	// 수급 데이터 조회
	GetFlowFactors(ctx context.Context, symbol string) (*FlowFactors, error)

	// 이벤트 데이터 조회
	GetEventFactors(ctx context.Context, symbol string) (*EventFactors, error)
}

// FactorScoreRepository 팩터 점수 저장소 (계산 결과 저장용)
type FactorScoreRepository interface {
	// 팩터 점수 저장
	SaveFactorScores(ctx context.Context, scores *FactorScoreRecord) error

	// 팩터 점수 조회
	GetFactorScores(ctx context.Context, symbol string, date time.Time) (*FactorScoreRecord, error)

	// 최신 팩터 점수 조회
	GetLatestFactorScores(ctx context.Context, symbol string) (*FactorScoreRecord, error)

	// 날짜별 팩터 점수 목록
	ListFactorScoresByDate(ctx context.Context, date time.Time) ([]*FactorScoreRecord, error)
}

// FactorScoreRecord 팩터 점수 기록
type FactorScoreRecord struct {
	Symbol     string    `json:"symbol"`
	CalcDate   time.Time `json:"calc_date"`
	Momentum   float64   `json:"momentum"`   // -1.0 ~ 1.0
	Technical  float64   `json:"technical"`  // -1.0 ~ 1.0
	Value      float64   `json:"value"`      // -1.0 ~ 1.0
	Quality    float64   `json:"quality"`    // -1.0 ~ 1.0
	Flow       float64   `json:"flow"`       // -1.0 ~ 1.0
	Event      float64   `json:"event"`      // -1.0 ~ 1.0
	TotalScore float64   `json:"total_score"`
	UpdatedAt  time.Time `json:"updated_at"`
}
