package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// DisclosureRepository PostgreSQL 공시 저장소 (data.disclosures)
type DisclosureRepository struct {
	pool *postgres.Pool
}

// NewDisclosureRepository 저장소 생성
func NewDisclosureRepository(pool *postgres.Pool) *DisclosureRepository {
	return &DisclosureRepository{pool: pool}
}

// Save 공시 저장 (중복 시 무시)
func (r *DisclosureRepository) Save(ctx context.Context, disc *fetcher.Disclosure) error {
	// dart_rcept_no로 중복 체크
	if disc.DartRceptNo != nil {
		exists, err := r.ExistsByDartRceptNo(ctx, *disc.DartRceptNo)
		if err != nil {
			return err
		}
		if exists {
			return nil // 중복 무시
		}
	}

	query := `
		INSERT INTO data.disclosures
			(stock_code, disclosed_at, title, category, subcategory, content, url, dart_rcept_no)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		disc.StockCode, disc.DisclosedAt, disc.Title,
		disc.Category, disc.Subcategory, disc.Content,
		disc.URL, disc.DartRceptNo,
	)
	if err != nil {
		return fmt.Errorf("save disclosure: %w", err)
	}

	return nil
}

// SaveBatch 공시 일괄 저장
func (r *DisclosureRepository) SaveBatch(ctx context.Context, discs []*fetcher.Disclosure) (int, error) {
	if len(discs) == 0 {
		return 0, nil
	}

	// 기존 dart_rcept_no 목록 조회
	existingMap := make(map[string]bool)
	for _, disc := range discs {
		if disc.DartRceptNo != nil {
			existingMap[*disc.DartRceptNo] = false
		}
	}

	if len(existingMap) > 0 {
		// 기존 공시 확인
		rceptNos := make([]string, 0, len(existingMap))
		for rceptNo := range existingMap {
			rceptNos = append(rceptNos, rceptNo)
		}

		query := `SELECT dart_rcept_no FROM data.disclosures WHERE dart_rcept_no = ANY($1)`
		rows, err := r.pool.Query(ctx, query, rceptNos)
		if err != nil {
			return 0, fmt.Errorf("check existing disclosures: %w", err)
		}

		for rows.Next() {
			var rceptNo string
			if err := rows.Scan(&rceptNo); err != nil {
				rows.Close()
				return 0, fmt.Errorf("scan existing: %w", err)
			}
			existingMap[rceptNo] = true
		}
		rows.Close()
	}

	// 새 공시만 저장
	batch := &pgx.Batch{}
	query := `
		INSERT INTO data.disclosures
			(stock_code, disclosed_at, title, category, subcategory, content, url, dart_rcept_no)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	newCount := 0
	for _, disc := range discs {
		// 중복 체크
		if disc.DartRceptNo != nil {
			if existingMap[*disc.DartRceptNo] {
				continue
			}
		}

		batch.Queue(query,
			disc.StockCode, disc.DisclosedAt, disc.Title,
			disc.Category, disc.Subcategory, disc.Content,
			disc.URL, disc.DartRceptNo,
		)
		newCount++
	}

	if newCount == 0 {
		return 0, nil
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	count := 0
	for i := 0; i < newCount; i++ {
		_, err := br.Exec()
		if err != nil {
			return count, fmt.Errorf("batch save disclosure: %w", err)
		}
		count++
	}

	return count, nil
}

// GetByStock 종목별 공시 조회
func (r *DisclosureRepository) GetByStock(ctx context.Context, stockCode string, from, to time.Time) ([]*fetcher.Disclosure, error) {
	query := `
		SELECT id, stock_code, disclosed_at, title, category, subcategory, content, url, dart_rcept_no, created_at
		FROM data.disclosures
		WHERE stock_code = $1 AND disclosed_at >= $2 AND disclosed_at <= $3
		ORDER BY disclosed_at DESC
	`

	rows, err := r.pool.Query(ctx, query, stockCode, from, to)
	if err != nil {
		return nil, fmt.Errorf("query disclosures: %w", err)
	}
	defer rows.Close()

	var disclosures []*fetcher.Disclosure
	for rows.Next() {
		var disc fetcher.Disclosure
		if err := rows.Scan(
			&disc.ID, &disc.StockCode, &disc.DisclosedAt, &disc.Title,
			&disc.Category, &disc.Subcategory, &disc.Content,
			&disc.URL, &disc.DartRceptNo, &disc.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan disclosure: %w", err)
		}
		disclosures = append(disclosures, &disc)
	}

	return disclosures, rows.Err()
}

// GetByCategory 카테고리별 공시 조회
func (r *DisclosureRepository) GetByCategory(ctx context.Context, category string, from, to time.Time) ([]*fetcher.Disclosure, error) {
	query := `
		SELECT id, stock_code, disclosed_at, title, category, subcategory, content, url, dart_rcept_no, created_at
		FROM data.disclosures
		WHERE category = $1 AND disclosed_at >= $2 AND disclosed_at <= $3
		ORDER BY disclosed_at DESC
	`

	rows, err := r.pool.Query(ctx, query, category, from, to)
	if err != nil {
		return nil, fmt.Errorf("query disclosures: %w", err)
	}
	defer rows.Close()

	var disclosures []*fetcher.Disclosure
	for rows.Next() {
		var disc fetcher.Disclosure
		if err := rows.Scan(
			&disc.ID, &disc.StockCode, &disc.DisclosedAt, &disc.Title,
			&disc.Category, &disc.Subcategory, &disc.Content,
			&disc.URL, &disc.DartRceptNo, &disc.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan disclosure: %w", err)
		}
		disclosures = append(disclosures, &disc)
	}

	return disclosures, rows.Err()
}

// GetRecent 최근 공시 조회
func (r *DisclosureRepository) GetRecent(ctx context.Context, limit int) ([]*fetcher.Disclosure, error) {
	query := `
		SELECT id, stock_code, disclosed_at, title, category, subcategory, content, url, dart_rcept_no, created_at
		FROM data.disclosures
		ORDER BY disclosed_at DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query disclosures: %w", err)
	}
	defer rows.Close()

	var disclosures []*fetcher.Disclosure
	for rows.Next() {
		var disc fetcher.Disclosure
		if err := rows.Scan(
			&disc.ID, &disc.StockCode, &disc.DisclosedAt, &disc.Title,
			&disc.Category, &disc.Subcategory, &disc.Content,
			&disc.URL, &disc.DartRceptNo, &disc.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan disclosure: %w", err)
		}
		disclosures = append(disclosures, &disc)
	}

	return disclosures, rows.Err()
}

// ExistsByDartRceptNo DART 접수번호로 존재 여부 확인
func (r *DisclosureRepository) ExistsByDartRceptNo(ctx context.Context, dartRceptNo string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM data.disclosures WHERE dart_rcept_no = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, dartRceptNo).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check disclosure exists: %w", err)
	}

	return exists, nil
}
