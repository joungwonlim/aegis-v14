package fetcher

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
)

// FetchLogRepository PostgreSQL 구현
type FetchLogRepository struct {
	db *pgxpool.Pool
}

// NewFetchLogRepository 생성자
func NewFetchLogRepository(db *pgxpool.Pool) *FetchLogRepository {
	return &FetchLogRepository{db: db}
}

// Create 로그 생성 (실행 시작 시)
func (r *FetchLogRepository) Create(ctx context.Context, log *fetcher.FetchLog) (*fetcher.FetchLog, error) {
	query := `
		INSERT INTO data.fetch_logs (
			job_type, source, target_table, records_fetched,
			records_inserted, records_updated, status, error_message,
			started_at, finished_at, duration_ms
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		log.JobType,
		log.Source,
		log.TargetTable,
		log.RecordsFetched,
		log.RecordsInserted,
		log.RecordsUpdated,
		log.Status,
		log.ErrorMessage,
		log.StartedAt,
		log.FinishedAt,
		log.DurationMs,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("create fetch log: %w", err)
	}

	return log, nil
}

// Update 로그 업데이트 (실행 완료/실패 시)
func (r *FetchLogRepository) Update(ctx context.Context, log *fetcher.FetchLog) error {
	query := `
		UPDATE data.fetch_logs
		SET records_fetched = $1,
		    records_inserted = $2,
		    records_updated = $3,
		    status = $4,
		    error_message = $5,
		    finished_at = $6,
		    duration_ms = $7
		WHERE id = $8
	`

	result, err := r.db.Exec(ctx, query,
		log.RecordsFetched,
		log.RecordsInserted,
		log.RecordsUpdated,
		log.Status,
		log.ErrorMessage,
		log.FinishedAt,
		log.DurationMs,
		log.ID,
	)

	if err != nil {
		return fmt.Errorf("update fetch log: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("fetch log not found: id=%d", log.ID)
	}

	return nil
}

// GetByID ID로 로그 조회
func (r *FetchLogRepository) GetByID(ctx context.Context, id int) (*fetcher.FetchLog, error) {
	query := `
		SELECT id, job_type, source, target_table, records_fetched,
		       records_inserted, records_updated, status, error_message,
		       started_at, finished_at, duration_ms, created_at
		FROM data.fetch_logs
		WHERE id = $1
	`

	var log fetcher.FetchLog
	err := r.db.QueryRow(ctx, query, id).Scan(
		&log.ID,
		&log.JobType,
		&log.Source,
		&log.TargetTable,
		&log.RecordsFetched,
		&log.RecordsInserted,
		&log.RecordsUpdated,
		&log.Status,
		&log.ErrorMessage,
		&log.StartedAt,
		&log.FinishedAt,
		&log.DurationMs,
		&log.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get fetch log by id: %w", err)
	}

	return &log, nil
}

// GetRecent 최근 로그 조회
func (r *FetchLogRepository) GetRecent(ctx context.Context, limit int) ([]*fetcher.FetchLog, error) {
	query := `
		SELECT id, job_type, source, target_table, records_fetched,
		       records_inserted, records_updated, status, error_message,
		       started_at, finished_at, duration_ms, created_at
		FROM data.fetch_logs
		ORDER BY started_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent fetch logs: %w", err)
	}
	defer rows.Close()

	var logs []*fetcher.FetchLog
	for rows.Next() {
		var log fetcher.FetchLog
		err := rows.Scan(
			&log.ID,
			&log.JobType,
			&log.Source,
			&log.TargetTable,
			&log.RecordsFetched,
			&log.RecordsInserted,
			&log.RecordsUpdated,
			&log.Status,
			&log.ErrorMessage,
			&log.StartedAt,
			&log.FinishedAt,
			&log.DurationMs,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan fetch log: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// GetByJobType job_type으로 로그 조회
func (r *FetchLogRepository) GetByJobType(ctx context.Context, jobType string, limit int) ([]*fetcher.FetchLog, error) {
	query := `
		SELECT id, job_type, source, target_table, records_fetched,
		       records_inserted, records_updated, status, error_message,
		       started_at, finished_at, duration_ms, created_at
		FROM data.fetch_logs
		WHERE job_type = $1
		ORDER BY started_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, jobType, limit)
	if err != nil {
		return nil, fmt.Errorf("get fetch logs by job type: %w", err)
	}
	defer rows.Close()

	var logs []*fetcher.FetchLog
	for rows.Next() {
		var log fetcher.FetchLog
		err := rows.Scan(
			&log.ID,
			&log.JobType,
			&log.Source,
			&log.TargetTable,
			&log.RecordsFetched,
			&log.RecordsInserted,
			&log.RecordsUpdated,
			&log.Status,
			&log.ErrorMessage,
			&log.StartedAt,
			&log.FinishedAt,
			&log.DurationMs,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan fetch log: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// GetByStatus status로 로그 조회
func (r *FetchLogRepository) GetByStatus(ctx context.Context, status string, limit int) ([]*fetcher.FetchLog, error) {
	query := `
		SELECT id, job_type, source, target_table, records_fetched,
		       records_inserted, records_updated, status, error_message,
		       started_at, finished_at, duration_ms, created_at
		FROM data.fetch_logs
		WHERE status = $1
		ORDER BY started_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("get fetch logs by status: %w", err)
	}
	defer rows.Close()

	var logs []*fetcher.FetchLog
	for rows.Next() {
		var log fetcher.FetchLog
		err := rows.Scan(
			&log.ID,
			&log.JobType,
			&log.Source,
			&log.TargetTable,
			&log.RecordsFetched,
			&log.RecordsInserted,
			&log.RecordsUpdated,
			&log.Status,
			&log.ErrorMessage,
			&log.StartedAt,
			&log.FinishedAt,
			&log.DurationMs,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan fetch log: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}
