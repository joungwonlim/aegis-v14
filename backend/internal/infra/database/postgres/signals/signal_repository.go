package signals

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/signals"
)

// SignalRepository 신호 저장소 구현
type SignalRepository struct {
	pool *pgxpool.Pool
}

// NewSignalRepository 새 리포지토리 생성
func NewSignalRepository(pool *pgxpool.Pool) *SignalRepository {
	return &SignalRepository{pool: pool}
}

// SaveSnapshot 스냅샷 저장
func (r *SignalRepository) SaveSnapshot(ctx context.Context, snapshot *signals.SignalSnapshot) error {
	// JSON 직렬화
	buySignalsJSON, err := json.Marshal(snapshot.BuySignals)
	if err != nil {
		return err
	}
	sellSignalsJSON, err := json.Marshal(snapshot.SellSignals)
	if err != nil {
		return err
	}
	statsJSON, err := json.Marshal(snapshot.Stats)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO signals.snapshots (
			snapshot_id, universe_id, generated_at,
			total_count, buy_signals, sell_signals, stats
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (snapshot_id) DO UPDATE SET
			universe_id = EXCLUDED.universe_id,
			generated_at = EXCLUDED.generated_at,
			total_count = EXCLUDED.total_count,
			buy_signals = EXCLUDED.buy_signals,
			sell_signals = EXCLUDED.sell_signals,
			stats = EXCLUDED.stats
	`

	_, err = r.pool.Exec(ctx, query,
		snapshot.SnapshotID,
		snapshot.UniverseID,
		snapshot.GeneratedAt,
		snapshot.TotalCount,
		buySignalsJSON,
		sellSignalsJSON,
		statsJSON,
	)

	return err
}

// GetLatestSnapshot 최신 스냅샷 조회
func (r *SignalRepository) GetLatestSnapshot(ctx context.Context) (*signals.SignalSnapshot, error) {
	query := `
		SELECT snapshot_id, universe_id, generated_at,
			   total_count, buy_signals, sell_signals, stats
		FROM signals.snapshots
		ORDER BY generated_at DESC
		LIMIT 1
	`

	return r.scanSnapshot(ctx, query)
}

// GetSnapshotByID 특정 스냅샷 조회
func (r *SignalRepository) GetSnapshotByID(ctx context.Context, snapshotID string) (*signals.SignalSnapshot, error) {
	query := `
		SELECT snapshot_id, universe_id, generated_at,
			   total_count, buy_signals, sell_signals, stats
		FROM signals.snapshots
		WHERE snapshot_id = $1
	`

	return r.scanSnapshotWithArg(ctx, query, snapshotID)
}

// ListSnapshots 스냅샷 목록 조회 (시간 범위)
func (r *SignalRepository) ListSnapshots(ctx context.Context, from, to time.Time) ([]*signals.SignalSnapshot, error) {
	query := `
		SELECT snapshot_id, universe_id, generated_at,
			   total_count, buy_signals, sell_signals, stats
		FROM signals.snapshots
		WHERE generated_at >= $1 AND generated_at <= $2
		ORDER BY generated_at DESC
	`

	rows, err := r.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := make([]*signals.SignalSnapshot, 0)
	for rows.Next() {
		snapshot, err := r.scanSnapshotRow(rows)
		if err != nil {
			continue
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// GetSignalBySymbol 특정 종목의 신호 조회
func (r *SignalRepository) GetSignalBySymbol(ctx context.Context, snapshotID, symbol string) (*signals.Signal, error) {
	// 스냅샷 조회
	snapshot, err := r.GetSnapshotByID(ctx, snapshotID)
	if err != nil {
		return nil, err
	}

	// Buy signals에서 검색
	for _, sig := range snapshot.BuySignals {
		if sig.Symbol == symbol {
			return &sig, nil
		}
	}

	// Sell signals에서 검색
	for _, sig := range snapshot.SellSignals {
		if sig.Symbol == symbol {
			return &sig, nil
		}
	}

	return nil, signals.ErrSignalNotFound
}

// Helper methods

func (r *SignalRepository) scanSnapshot(ctx context.Context, query string) (*signals.SignalSnapshot, error) {
	row := r.pool.QueryRow(ctx, query)
	return r.scanSnapshotFromRow(row)
}

func (r *SignalRepository) scanSnapshotWithArg(ctx context.Context, query string, arg interface{}) (*signals.SignalSnapshot, error) {
	row := r.pool.QueryRow(ctx, query, arg)
	return r.scanSnapshotFromRow(row)
}

func (r *SignalRepository) scanSnapshotFromRow(row pgx.Row) (*signals.SignalSnapshot, error) {
	var snapshot signals.SignalSnapshot
	var buySignalsJSON, sellSignalsJSON, statsJSON []byte

	err := row.Scan(
		&snapshot.SnapshotID,
		&snapshot.UniverseID,
		&snapshot.GeneratedAt,
		&snapshot.TotalCount,
		&buySignalsJSON,
		&sellSignalsJSON,
		&statsJSON,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, signals.ErrSnapshotNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal(buySignalsJSON, &snapshot.BuySignals); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(sellSignalsJSON, &snapshot.SellSignals); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(statsJSON, &snapshot.Stats); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (r *SignalRepository) scanSnapshotRow(rows pgx.Rows) (*signals.SignalSnapshot, error) {
	var snapshot signals.SignalSnapshot
	var buySignalsJSON, sellSignalsJSON, statsJSON []byte

	err := rows.Scan(
		&snapshot.SnapshotID,
		&snapshot.UniverseID,
		&snapshot.GeneratedAt,
		&snapshot.TotalCount,
		&buySignalsJSON,
		&sellSignalsJSON,
		&statsJSON,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(buySignalsJSON, &snapshot.BuySignals); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(sellSignalsJSON, &snapshot.SellSignals); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(statsJSON, &snapshot.Stats); err != nil {
		return nil, err
	}

	return &snapshot, nil
}
