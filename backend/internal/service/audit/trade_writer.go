package audit

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/audit"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// =============================================================================
// AuditTradeWriter Implementation
// =============================================================================

// TradeWriter implements execution.AuditTradeWriter
// It saves trades to audit.trade_history when ExitEvents are created
type TradeWriter struct {
	repo audit.Repository
}

// NewTradeWriter creates a new TradeWriter
func NewTradeWriter(repo audit.Repository) *TradeWriter {
	return &TradeWriter{repo: repo}
}

// SaveExitTrade saves a trade to audit.trade_history when an exit event is created
func (w *TradeWriter) SaveExitTrade(ctx context.Context, event *execution.ExitEvent, entryDate time.Time) error {
	// Calculate hold days
	holdDays := int(event.ExitTS.Sub(entryDate).Hours() / 24)
	if holdDays < 0 {
		holdDays = 0
	}

	// Convert ExitEvent to audit.Trade
	trade := &audit.Trade{
		Symbol:     event.Symbol,
		Side:       "SELL", // ExitEvent is always a sell
		Quantity:   int(event.ExitQty),
		Price:      event.ExitAvgPrice.IntPart(), // Exit price
		PnL:        event.RealizedPnl.InexactFloat64(),
		PnLPercent: event.RealizedPnlPct / 100.0, // Convert from % to decimal
		EntryDate:  entryDate,
		ExitDate:   event.ExitTS,
		HoldDays:   holdDays,
		ExitReason: event.ExitReasonCode,
	}

	// Save to audit.trade_history
	if err := w.repo.SaveTradeHistory(ctx, trade); err != nil {
		return err
	}

	log.Info().
		Str("symbol", event.Symbol).
		Str("exit_reason", event.ExitReasonCode).
		Float64("pnl", trade.PnL).
		Float64("pnl_pct", trade.PnLPercent*100).
		Int("hold_days", holdDays).
		Msg("✅ Trade saved to audit.trade_history")

	return nil
}
