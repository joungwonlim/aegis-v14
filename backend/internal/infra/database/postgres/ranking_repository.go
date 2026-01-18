package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
)

// RankingRepository PostgreSQL 순위 저장소 구현
type RankingRepository struct {
	pool *pgxpool.Pool
}

// NewRankingRepository 순위 저장소 생성
func NewRankingRepository(pool *pgxpool.Pool) *RankingRepository {
	return &RankingRepository{pool: pool}
}

// SaveBatch 순위 데이터 배치 저장 (기존 데이터 삭제 후 저장)
func (r *RankingRepository) SaveBatch(ctx context.Context, category, market string, rankings []*fetcher.RankingStock) error {
	if len(rankings) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 기존 데이터 삭제 (같은 category + market + 최근 10분 이내 데이터)
	deleteQuery := `
		DELETE FROM data.stock_rankings
		WHERE category = $1 AND market = $2
		  AND collected_at >= NOW() - INTERVAL '10 minutes'
	`
	_, err = tx.Exec(ctx, deleteQuery, category, market)
	if err != nil {
		return fmt.Errorf("delete old rankings: %w", err)
	}

	// 새 데이터 삽입
	insertQuery := `
		INSERT INTO data.stock_rankings (
			category, market, rank, stock_code, stock_name,
			current_price, change_rate, volume, trading_value, high_price, low_price,
			foreign_net_value, inst_net_value, volume_surge_rate, high_52week, market_cap,
			collected_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17
		)
	`

	collectedAt := time.Now()
	for _, ranking := range rankings {
		_, err = tx.Exec(ctx, insertQuery,
			category, market, ranking.Rank, ranking.StockCode, ranking.StockName,
			ranking.CurrentPrice, ranking.ChangeRate, ranking.Volume, ranking.TradingValue, ranking.HighPrice, ranking.LowPrice,
			ranking.ForeignNetValue, ranking.InstNetValue, ranking.VolumeSurgeRate, ranking.High52Week, ranking.MarketCap,
			collectedAt,
		)
		if err != nil {
			return fmt.Errorf("insert ranking: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	log.Debug().
		Str("category", category).
		Str("market", market).
		Int("count", len(rankings)).
		Msg("Saved ranking batch")

	return nil
}

// GetLatest 최신 순위 조회
func (r *RankingRepository) GetLatest(ctx context.Context, category, market string, limit int) ([]*fetcher.RankingStock, error) {
	// 최신 collected_at 조회
	var latestCollectedAt time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT MAX(collected_at)
		FROM data.stock_rankings
		WHERE category = $1 AND market = $2
	`, category, market).Scan(&latestCollectedAt)
	if err != nil {
		return nil, fmt.Errorf("get latest collected_at: %w", err)
	}

	// 최신 데이터 조회
	query := `
		SELECT
			id, category, market, rank, stock_code, stock_name,
			current_price, change_rate, volume, trading_value, high_price, low_price,
			foreign_net_value, inst_net_value, volume_surge_rate, high_52week, market_cap,
			collected_at, created_at
		FROM data.stock_rankings
		WHERE category = $1 AND market = $2 AND collected_at = $3
		ORDER BY rank
		LIMIT $4
	`

	rows, err := r.pool.Query(ctx, query, category, market, latestCollectedAt, limit)
	if err != nil {
		return nil, fmt.Errorf("query rankings: %w", err)
	}
	defer rows.Close()

	var rankings []*fetcher.RankingStock
	for rows.Next() {
		var r fetcher.RankingStock
		err := rows.Scan(
			&r.ID, &r.Category, &r.Market, &r.Rank, &r.StockCode, &r.StockName,
			&r.CurrentPrice, &r.ChangeRate, &r.Volume, &r.TradingValue, &r.HighPrice, &r.LowPrice,
			&r.ForeignNetValue, &r.InstNetValue, &r.VolumeSurgeRate, &r.High52Week, &r.MarketCap,
			&r.CollectedAt, &r.CreatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan ranking row")
			continue
		}
		rankings = append(rankings, &r)
	}

	return rankings, nil
}

// GetLatestCollectedAt 최신 수집 시각 조회
func (r *RankingRepository) GetLatestCollectedAt(ctx context.Context, category, market string) (*time.Time, error) {
	var collectedAt time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT MAX(collected_at)
		FROM data.stock_rankings
		WHERE category = $1 AND market = $2
	`, category, market).Scan(&collectedAt)
	if err != nil {
		return nil, fmt.Errorf("get latest collected_at: %w", err)
	}
	return &collectedAt, nil
}

// DeleteOld 오래된 순위 데이터 삭제 (1일 이상 된 데이터)
func (r *RankingRepository) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := r.pool.Exec(ctx, `
		DELETE FROM data.stock_rankings
		WHERE created_at < $1
	`, olderThan)
	if err != nil {
		return 0, fmt.Errorf("delete old rankings: %w", err)
	}

	deleted := result.RowsAffected()
	if deleted > 0 {
		log.Info().
			Int64("deleted", deleted).
			Time("older_than", olderThan).
			Msg("Deleted old rankings")
	}

	return deleted, nil
}
