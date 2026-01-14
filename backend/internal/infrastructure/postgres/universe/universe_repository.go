package universe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/universe"
)

// UniverseRepository implements universe.UniverseRepository
type UniverseRepository struct {
	db *pgxpool.Pool
}

// NewUniverseRepository creates a new universe repository
func NewUniverseRepository(db *pgxpool.Pool) *UniverseRepository {
	return &UniverseRepository{db: db}
}

// SaveSnapshot saves a universe snapshot
func (r *UniverseRepository) SaveSnapshot(ctx context.Context, snapshot *universe.UniverseSnapshot) error {
	// Serialize JSONB fields
	holdingsJSON, err := json.Marshal(snapshot.Holdings)
	if err != nil {
		return fmt.Errorf("marshal holdings: %w", err)
	}

	watchlistJSON, err := json.Marshal(snapshot.Watchlist)
	if err != nil {
		return fmt.Errorf("marshal watchlist: %w", err)
	}

	rankingsJSON, err := json.Marshal(snapshot.Rankings)
	if err != nil {
		return fmt.Errorf("marshal rankings: %w", err)
	}

	filterStatsJSON, err := json.Marshal(snapshot.FilterStats)
	if err != nil {
		return fmt.Errorf("marshal filter_stats: %w", err)
	}

	query := `
		INSERT INTO market.universe_snapshots (
			snapshot_id, generated_at, total_count,
			holdings, watchlist, rankings, filter_stats, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (snapshot_id) DO UPDATE SET
			generated_at = EXCLUDED.generated_at,
			total_count = EXCLUDED.total_count,
			holdings = EXCLUDED.holdings,
			watchlist = EXCLUDED.watchlist,
			rankings = EXCLUDED.rankings,
			filter_stats = EXCLUDED.filter_stats,
			created_at = EXCLUDED.created_at
	`

	_, err = r.db.Exec(ctx, query,
		snapshot.SnapshotID,
		snapshot.GeneratedAt,
		snapshot.TotalCount,
		holdingsJSON,
		watchlistJSON,
		rankingsJSON,
		filterStatsJSON,
	)

	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}

	return nil
}

// GetLatestSnapshot retrieves the latest universe snapshot
func (r *UniverseRepository) GetLatestSnapshot(ctx context.Context) (*universe.UniverseSnapshot, error) {
	query := `
		SELECT snapshot_id, generated_at, total_count,
		       holdings, watchlist, rankings, filter_stats
		FROM market.universe_snapshots
		ORDER BY generated_at DESC
		LIMIT 1
	`

	var snapshot universe.UniverseSnapshot
	var holdingsJSON, watchlistJSON, rankingsJSON, filterStatsJSON []byte

	err := r.db.QueryRow(ctx, query).Scan(
		&snapshot.SnapshotID,
		&snapshot.GeneratedAt,
		&snapshot.TotalCount,
		&holdingsJSON,
		&watchlistJSON,
		&rankingsJSON,
		&filterStatsJSON,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, universe.ErrSnapshotNotFound
		}
		return nil, fmt.Errorf("query latest snapshot: %w", err)
	}

	// Deserialize JSONB fields
	if err := json.Unmarshal(holdingsJSON, &snapshot.Holdings); err != nil {
		return nil, fmt.Errorf("unmarshal holdings: %w", err)
	}

	if err := json.Unmarshal(watchlistJSON, &snapshot.Watchlist); err != nil {
		return nil, fmt.Errorf("unmarshal watchlist: %w", err)
	}

	if err := json.Unmarshal(rankingsJSON, &snapshot.Rankings); err != nil {
		return nil, fmt.Errorf("unmarshal rankings: %w", err)
	}

	if err := json.Unmarshal(filterStatsJSON, &snapshot.FilterStats); err != nil {
		return nil, fmt.Errorf("unmarshal filter_stats: %w", err)
	}

	return &snapshot, nil
}

// GetSnapshotByID retrieves a snapshot by ID
func (r *UniverseRepository) GetSnapshotByID(ctx context.Context, snapshotID string) (*universe.UniverseSnapshot, error) {
	query := `
		SELECT snapshot_id, generated_at, total_count,
		       holdings, watchlist, rankings, filter_stats
		FROM market.universe_snapshots
		WHERE snapshot_id = $1
	`

	var snapshot universe.UniverseSnapshot
	var holdingsJSON, watchlistJSON, rankingsJSON, filterStatsJSON []byte

	err := r.db.QueryRow(ctx, query, snapshotID).Scan(
		&snapshot.SnapshotID,
		&snapshot.GeneratedAt,
		&snapshot.TotalCount,
		&holdingsJSON,
		&watchlistJSON,
		&rankingsJSON,
		&filterStatsJSON,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, universe.ErrSnapshotNotFound
		}
		return nil, fmt.Errorf("query snapshot by id: %w", err)
	}

	// Deserialize JSONB fields
	if err := json.Unmarshal(holdingsJSON, &snapshot.Holdings); err != nil {
		return nil, fmt.Errorf("unmarshal holdings: %w", err)
	}

	if err := json.Unmarshal(watchlistJSON, &snapshot.Watchlist); err != nil {
		return nil, fmt.Errorf("unmarshal watchlist: %w", err)
	}

	if err := json.Unmarshal(rankingsJSON, &snapshot.Rankings); err != nil {
		return nil, fmt.Errorf("unmarshal rankings: %w", err)
	}

	if err := json.Unmarshal(filterStatsJSON, &snapshot.FilterStats); err != nil {
		return nil, fmt.Errorf("unmarshal filter_stats: %w", err)
	}

	return &snapshot, nil
}

// ListSnapshots lists snapshots within a time range
func (r *UniverseRepository) ListSnapshots(ctx context.Context, from, to time.Time) ([]*universe.UniverseSnapshot, error) {
	query := `
		SELECT snapshot_id, generated_at, total_count,
		       holdings, watchlist, rankings, filter_stats
		FROM market.universe_snapshots
		WHERE generated_at BETWEEN $1 AND $2
		ORDER BY generated_at DESC
	`

	rows, err := r.db.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("query snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*universe.UniverseSnapshot
	for rows.Next() {
		var snapshot universe.UniverseSnapshot
		var holdingsJSON, watchlistJSON, rankingsJSON, filterStatsJSON []byte

		err := rows.Scan(
			&snapshot.SnapshotID,
			&snapshot.GeneratedAt,
			&snapshot.TotalCount,
			&holdingsJSON,
			&watchlistJSON,
			&rankingsJSON,
			&filterStatsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}

		// Deserialize JSONB fields
		if err := json.Unmarshal(holdingsJSON, &snapshot.Holdings); err != nil {
			return nil, fmt.Errorf("unmarshal holdings: %w", err)
		}

		if err := json.Unmarshal(watchlistJSON, &snapshot.Watchlist); err != nil {
			return nil, fmt.Errorf("unmarshal watchlist: %w", err)
		}

		if err := json.Unmarshal(rankingsJSON, &snapshot.Rankings); err != nil {
			return nil, fmt.Errorf("unmarshal rankings: %w", err)
		}

		if err := json.Unmarshal(filterStatsJSON, &snapshot.FilterStats); err != nil {
			return nil, fmt.Errorf("unmarshal filter_stats: %w", err)
		}

		snapshots = append(snapshots, &snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return snapshots, nil
}
