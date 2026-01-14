package ranking

import "errors"

var (
	// ErrSnapshotNotFound 스냅샷을 찾을 수 없음
	ErrSnapshotNotFound = errors.New("ranking snapshot not found")

	// ErrRankNotFound 순위를 찾을 수 없음
	ErrRankNotFound = errors.New("rank not found")

	// ErrInvalidCriteria 잘못된 기준
	ErrInvalidCriteria = errors.New("invalid ranking criteria")

	// ErrRiskDataMissing 리스크 데이터 누락
	ErrRiskDataMissing = errors.New("risk data missing")

	// ErrSignalsNotReady Signals가 준비되지 않음
	ErrSignalsNotReady = errors.New("signals not ready")

	// ErrNoValidStocks 유효한 종목이 없음
	ErrNoValidStocks = errors.New("no valid stocks after filtering")
)
