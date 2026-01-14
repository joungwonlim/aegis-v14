package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// syncHoldings syncs holdings from KIS and detects ExitEvents
func (s *Service) syncHoldings(ctx context.Context) error {
	// 1. Fetch holdings from KIS
	kisHoldings, err := s.kisAdapter.GetHoldings(ctx, s.accountID)
	if err != nil {
		return fmt.Errorf("fetch holdings: %w", err)
	}

	// 2. Upsert holdings to DB
	var currHoldings []*execution.Holding
	for _, kh := range kisHoldings {
		holding := &execution.Holding{
			AccountID:    kh.AccountID,
			Symbol:       kh.Symbol,
			Qty:          kh.Qty,
			AvgPrice:     kh.AvgPrice,
			CurrentPrice: kh.CurrentPrice,
			Pnl:          kh.Pnl,
			PnlPct:       kh.PnlPct,
			UpdatedTS:    time.Now(),
			Raw:          kh.Raw,
		}

		if err := s.holdingRepo.UpsertHolding(ctx, holding); err != nil {
			log.Error().Err(err).Str("symbol", kh.Symbol).Msg("Failed to upsert holding")
			continue
		}

		currHoldings = append(currHoldings, holding)
	}

	log.Debug().Int("count", len(currHoldings)).Msg("Holdings synced")

	// 3. Detect and create ExitEvents (qty: N → 0)
	if err := s.detectAndCreateExitEvents(ctx, s.prevHoldings, currHoldings); err != nil {
		log.Error().Err(err).Msg("Failed to detect exit events")
		// Don't return error - holdings sync succeeded
	}

	// 4. Update previous holdings snapshot
	s.prevHoldings = currHoldings

	return nil
}

// detectAndCreateExitEvents detects holdings that went to zero and creates ExitEvents
func (s *Service) detectAndCreateExitEvents(ctx context.Context, prevHoldings, currHoldings []*execution.Holding) error {
	// Build maps for quick lookup
	prevMap := make(map[string]*execution.Holding)
	for _, h := range prevHoldings {
		prevMap[h.Symbol] = h
	}

	currMap := make(map[string]*execution.Holding)
	for _, h := range currHoldings {
		currMap[h.Symbol] = h
	}

	// Detect: prev qty > 0, curr qty = 0 OR curr not exists
	for symbol, prev := range prevMap {
		if prev.Qty <= 0 {
			continue
		}

		curr, exists := currMap[symbol]

		// Case 1: qty: N → 0
		if exists && curr.Qty == 0 {
			if err := s.createExitEvent(ctx, prev.AccountID, symbol); err != nil {
				log.Error().
					Err(err).
					Str("symbol", symbol).
					Int64("prev_qty", prev.Qty).
					Msg("Failed to create exit event")
			}
		}

		// Case 2: holding disappeared (qty: N → not exists)
		if !exists {
			if err := s.createExitEvent(ctx, prev.AccountID, symbol); err != nil {
				log.Error().
					Err(err).
					Str("symbol", symbol).
					Int64("prev_qty", prev.Qty).
					Msg("Failed to create exit event (holding disappeared)")
			}
		}
	}

	return nil
}

// createExitEvent creates an ExitEvent for a position
func (s *Service) createExitEvent(ctx context.Context, accountID, symbol string) error {
	// 1. Load position by symbol
	position, err := s.positionRepo.GetPositionBySymbol(ctx, accountID, symbol, "OPEN")
	if err != nil {
		if err == execution.ErrPositionNotFound {
			log.Warn().
				Str("symbol", symbol).
				Msg("No open position found for cleared holding (orphan holding?)")
			return nil
		}
		return fmt.Errorf("get position: %w", err)
	}

	// 2. Check if exit event already exists (idempotency)
	exists, err := s.exitEventRepo.ExitEventExists(ctx, position.PositionID)
	if err != nil {
		return fmt.Errorf("check exit event exists: %w", err)
	}
	if exists {
		log.Debug().
			Str("position_id", position.PositionID.String()).
			Str("symbol", symbol).
			Msg("ExitEvent already exists, skipping")
		return nil
	}

	// 3. Determine exit reason and source
	exitReasonCode, source, intentID := s.determineExitReason(ctx, position.PositionID)

	// 4. Calculate exit average price
	exitAvgPrice := s.calculateExitAvgPrice(ctx, position.PositionID)

	// 5. Calculate realized PnL
	entryValue := position.AvgPrice.Mul(decimal.NewFromInt(position.Qty))
	exitValue := exitAvgPrice.Mul(decimal.NewFromInt(position.Qty))
	realizedPnl := exitValue.Sub(entryValue)

	realizedPnlPct := 0.0
	if !entryValue.IsZero() {
		realizedPnlPct, _ = realizedPnl.Div(entryValue).Mul(decimal.NewFromInt(100)).Float64()
	}

	// 6. Create ExitEvent
	exitEvent := &execution.ExitEvent{
		ExitEventID:    uuid.New(),
		PositionID:     position.PositionID,
		AccountID:      accountID,
		Symbol:         symbol,
		ExitTS:         time.Now(),
		ExitQty:        position.Qty,
		ExitAvgPrice:   exitAvgPrice,
		ExitReasonCode: exitReasonCode,
		Source:         source,
		IntentID:       intentID,
		ExitProfileID:  &position.ExitProfileID,
		RealizedPnl:    realizedPnl,
		RealizedPnlPct: realizedPnlPct,
		CreatedTS:      time.Now(),
	}

	if err := s.exitEventRepo.CreateExitEvent(ctx, exitEvent); err != nil {
		return fmt.Errorf("create exit event: %w", err)
	}

	log.Info().
		Str("exit_event_id", exitEvent.ExitEventID.String()).
		Str("position_id", position.PositionID.String()).
		Str("symbol", symbol).
		Str("exit_reason", exitReasonCode).
		Str("source", source).
		Str("realized_pnl", realizedPnl.StringFixed(2)).
		Float64("realized_pnl_pct", realizedPnlPct).
		Msg("ExitEvent created")

	return nil
}

// determineExitReason determines exit reason code and source
func (s *Service) determineExitReason(ctx context.Context, positionID uuid.UUID) (exitReasonCode string, source string, intentID *uuid.UUID) {
	// 1. Find recent EXIT intents for this position
	intentTypes := []string{"EXIT_PARTIAL", "EXIT_FULL"}
	statuses := []string{"SUBMITTED", "FILLED"}
	since := time.Now().Add(-1 * time.Hour) // Recent only (last 1 hour)

	intents, err := s.intentRepo.LoadIntentsForPosition(ctx, positionID, intentTypes, statuses, since)
	if err != nil || len(intents) == 0 {
		// No EXIT intent found → MANUAL or BROKER
		return execution.ExitReasonManual, execution.ExitSourceManual, nil
	}

	// 2. Take most recent EXIT intent (ordered by created_ts DESC)
	lastIntent := intents[0]

	// 3. Map intent reason_code to exit_reason_code
	source = execution.ExitSourceAutoExit
	exitReasonCode = lastIntent.ReasonCode // SL1, SL2, TP1, TP2, TP3, TRAIL, TIME
	intentID = &lastIntent.IntentID

	return exitReasonCode, source, intentID
}

// calculateExitAvgPrice calculates exit average price from fills
func (s *Service) calculateExitAvgPrice(ctx context.Context, positionID uuid.UUID) decimal.Decimal {
	// Load all EXIT fills for this position
	fills, err := s.fillRepo.LoadFillsForPosition(ctx, positionID, "EXIT")
	if err != nil || len(fills) == 0 {
		log.Warn().
			Str("position_id", positionID.String()).
			Msg("No exit fills found for position")
		return decimal.Zero
	}

	// Calculate weighted average
	totalValue := decimal.Zero
	totalQty := int64(0)
	for _, fill := range fills {
		totalValue = totalValue.Add(fill.Price.Mul(decimal.NewFromInt(fill.Qty)))
		totalQty += fill.Qty
	}

	if totalQty == 0 {
		return decimal.Zero
	}

	return totalValue.Div(decimal.NewFromInt(totalQty))
}
