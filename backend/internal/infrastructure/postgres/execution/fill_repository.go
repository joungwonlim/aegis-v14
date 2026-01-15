package execution

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// FillRepository implements execution.FillRepository
type FillRepository struct {
	db *pgxpool.Pool
}

// NewFillRepository creates a new fill repository
func NewFillRepository(db *pgxpool.Pool) *FillRepository {
	return &FillRepository{db: db}
}

// UpsertFill creates or updates a fill (idempotent by order_id, kis_exec_id, seq)
func (r *FillRepository) UpsertFill(ctx context.Context, fill *execution.Fill) error {
	query := `
		INSERT INTO trade.fills (
			fill_id, order_id, kis_exec_id, ts, qty, price, fee, tax, seq
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (order_id, kis_exec_id, seq) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query,
		fill.FillID,
		fill.OrderID,
		fill.KisExecID,
		fill.TS,
		fill.Qty,
		fill.Price,
		fill.Fee,
		fill.Tax,
		fill.Seq,
	)

	if err != nil {
		return fmt.Errorf("upsert fill: %w", err)
	}

	return nil
}

// LoadFills loads all fills for an order
func (r *FillRepository) LoadFills(ctx context.Context, orderID string) ([]*execution.Fill, error) {
	query := `
		SELECT fill_id, order_id, kis_exec_id, ts, qty, price, fee, tax, seq
		FROM trade.fills
		WHERE order_id = $1
		ORDER BY ts ASC, seq ASC
	`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("query fills: %w", err)
	}
	defer rows.Close()

	var fills []*execution.Fill
	for rows.Next() {
		var fill execution.Fill

		err := rows.Scan(
			&fill.FillID,
			&fill.OrderID,
			&fill.KisExecID,
			&fill.TS,
			&fill.Qty,
			&fill.Price,
			&fill.Fee,
			&fill.Tax,
			&fill.Seq,
		)
		if err != nil {
			return nil, fmt.Errorf("scan fill: %w", err)
		}

		fills = append(fills, &fill)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return fills, nil
}

// LoadFillsForPosition loads all fills for a position (via orders.intent_id → order_intents.position_id)
func (r *FillRepository) LoadFillsForPosition(ctx context.Context, positionID uuid.UUID, intentType string) ([]*execution.Fill, error) {
	query := `
		SELECT f.fill_id, f.order_id, f.kis_exec_id, f.ts, f.qty, f.price, f.fee, f.tax, f.seq
		FROM trade.fills f
		JOIN trade.orders o ON f.order_id = o.order_id
		JOIN trade.order_intents i ON o.intent_id = i.intent_id
		WHERE i.position_id = $1
		  AND i.intent_type LIKE $2
		ORDER BY f.ts ASC, f.seq ASC
	`

	intentTypePattern := "%" + intentType + "%"

	rows, err := r.db.Query(ctx, query, positionID, intentTypePattern)
	if err != nil {
		return nil, fmt.Errorf("query fills for position: %w", err)
	}
	defer rows.Close()

	var fills []*execution.Fill
	for rows.Next() {
		var fill execution.Fill

		err := rows.Scan(
			&fill.FillID,
			&fill.OrderID,
			&fill.KisExecID,
			&fill.TS,
			&fill.Qty,
			&fill.Price,
			&fill.Fee,
			&fill.Tax,
			&fill.Seq,
		)
		if err != nil {
			return nil, fmt.Errorf("scan fill: %w", err)
		}

		fills = append(fills, &fill)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return fills, nil
}

// LoadFillsSinceCursor loads fills since cursor (for sync)
func (r *FillRepository) LoadFillsSinceCursor(ctx context.Context, cursor execution.FillCursor) ([]*execution.Fill, error) {
	query := `
		SELECT fill_id, order_id, kis_exec_id, ts, qty, price, fee, tax, seq
		FROM trade.fills
		WHERE (ts > $1) OR (ts = $1 AND seq > $2)
		ORDER BY ts ASC, seq ASC
		LIMIT 1000
	`

	rows, err := r.db.Query(ctx, query, cursor.LastTS, cursor.LastSeq)
	if err != nil {
		return nil, fmt.Errorf("query fills since cursor: %w", err)
	}
	defer rows.Close()

	var fills []*execution.Fill
	for rows.Next() {
		var fill execution.Fill

		err := rows.Scan(
			&fill.FillID,
			&fill.OrderID,
			&fill.KisExecID,
			&fill.TS,
			&fill.Qty,
			&fill.Price,
			&fill.Fee,
			&fill.Tax,
			&fill.Seq,
		)
		if err != nil {
			return nil, fmt.Errorf("scan fill: %w", err)
		}

		fills = append(fills, &fill)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return fills, nil
}

// GetLastCursor retrieves the last sync cursor
func (r *FillRepository) GetLastCursor(ctx context.Context) (*execution.FillCursor, error) {
	query := `
		SELECT last_ts, last_seq
		FROM trade.fill_cursors
		WHERE id = 1
	`

	var cursor execution.FillCursor

	err := r.db.QueryRow(ctx, query).Scan(&cursor.LastTS, &cursor.LastSeq)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No cursor yet - return default
			return &execution.FillCursor{}, nil
		}
		return nil, fmt.Errorf("query cursor: %w", err)
	}

	return &cursor, nil
}

// SaveCursor saves the sync cursor
func (r *FillRepository) SaveCursor(ctx context.Context, cursor execution.FillCursor) error {
	query := `
		INSERT INTO trade.fill_cursors (id, last_ts, last_seq, updated_ts)
		VALUES (1, $1, $2, NOW())
		ON CONFLICT (id) DO UPDATE SET
			last_ts = EXCLUDED.last_ts,
			last_seq = EXCLUDED.last_seq,
			updated_ts = EXCLUDED.updated_ts
	`

	_, err := r.db.Exec(ctx, query, cursor.LastTS, cursor.LastSeq)
	if err != nil {
		return fmt.Errorf("save cursor: %w", err)
	}

	return nil
}

// GetRecentFills retrieves recent fills (for monitoring/API)
func (r *FillRepository) GetRecentFills(ctx context.Context, limit int) ([]*execution.Fill, error) {
	query := `
		SELECT fill_id, order_id, kis_exec_id, ts, qty, price, fee, tax, seq
		FROM trade.fills
		ORDER BY ts DESC, seq DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent fills: %w", err)
	}
	defer rows.Close()

	var fills []*execution.Fill
	for rows.Next() {
		var fill execution.Fill

		err := rows.Scan(
			&fill.FillID,
			&fill.OrderID,
			&fill.KisExecID,
			&fill.TS,
			&fill.Qty,
			&fill.Price,
			&fill.Fee,
			&fill.Tax,
			&fill.Seq,
		)
		if err != nil {
			return nil, fmt.Errorf("scan fill: %w", err)
		}

		fills = append(fills, &fill)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return fills, nil
}
