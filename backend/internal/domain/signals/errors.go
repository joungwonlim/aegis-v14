package signals

import "errors"

var (
	// ErrSnapshotNotFound 스냅샷을 찾을 수 없음
	ErrSnapshotNotFound = errors.New("signal snapshot not found")

	// ErrSignalNotFound 신호를 찾을 수 없음
	ErrSignalNotFound = errors.New("signal not found")

	// ErrInvalidCriteria 잘못된 기준
	ErrInvalidCriteria = errors.New("invalid signal criteria")

	// ErrFactorDataMissing 팩터 데이터 누락
	ErrFactorDataMissing = errors.New("factor data missing")

	// ErrUniverseNotReady Universe가 준비되지 않음
	ErrUniverseNotReady = errors.New("universe not ready")
)
