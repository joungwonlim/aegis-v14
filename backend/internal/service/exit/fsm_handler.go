package exit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// FSMHandler handles Exit FSM state transitions
type FSMHandler struct {
	stateRepo exit.PositionStateRepository
	posRepo   exit.PositionRepository
}

// NewFSMHandler creates a new FSM handler
func NewFSMHandler(
	stateRepo exit.PositionStateRepository,
	posRepo exit.PositionRepository,
) *FSMHandler {
	return &FSMHandler{
		stateRepo: stateRepo,
		posRepo:   posRepo,
	}
}

// HandleTP1Filled handles TP1 intent filled event (activates Stop Floor)
func (h *FSMHandler) HandleTP1Filled(ctx context.Context, positionID uuid.UUID, profile *exit.ExitProfile) error {
	// Get current position
	pos, err := h.posRepo.GetPosition(ctx, positionID)
	if err != nil {
		return fmt.Errorf("get position: %w", err)
	}

	// Calculate Stop Floor price
	// stop_floor_price = entry_price * (1 + be_profit_pct)
	beProfitPct := decimal.NewFromFloat(0.006) // 0.6% default
	if profile.Config.TP1.StopFloorProfit != nil {
		beProfitPct = decimal.NewFromFloat(*profile.Config.TP1.StopFloorProfit)
	}

	stopFloorPrice := pos.AvgPrice.Mul(decimal.NewFromInt(1).Add(beProfitPct))

	// Update state: OPEN → TP1_DONE, activate Stop Floor
	state, err := h.stateRepo.GetState(ctx, positionID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	state.Phase = exit.PhaseTP1Done
	state.StopFloorPrice = &stopFloorPrice

	err = h.stateRepo.UpsertState(ctx, state)
	if err != nil {
		return fmt.Errorf("update state: %w", err)
	}

	log.Info().
		Str("position_id", positionID.String()).
		Str("phase", exit.PhaseTP1Done).
		Str("stop_floor_price", stopFloorPrice.String()).
		Msg("TP1 filled: Stop Floor activated")

	return nil
}

// HandleTP2Filled handles TP2 intent filled event
func (h *FSMHandler) HandleTP2Filled(ctx context.Context, positionID uuid.UUID) error {
	// Update state: TP1_DONE → TP2_DONE
	err := h.stateRepo.UpdatePhase(ctx, positionID, exit.PhaseTP2Done)
	if err != nil {
		return fmt.Errorf("update phase: %w", err)
	}

	log.Info().
		Str("position_id", positionID.String()).
		Str("phase", exit.PhaseTP2Done).
		Msg("TP2 filled: Phase updated")

	return nil
}

// HandleTP3Filled handles TP3 intent filled event (starts Trailing)
func (h *FSMHandler) HandleTP3Filled(ctx context.Context, positionID uuid.UUID, currentPrice decimal.Decimal) error {
	// Update state: TP2_DONE → TRAILING_ACTIVE
	// Initialize HWM with current price
	state, err := h.stateRepo.GetState(ctx, positionID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	state.Phase = exit.PhaseTrailingActive
	state.HWMPrice = &currentPrice

	err = h.stateRepo.UpsertState(ctx, state)
	if err != nil {
		return fmt.Errorf("update state: %w", err)
	}

	log.Info().
		Str("position_id", positionID.String()).
		Str("phase", exit.PhaseTrailingActive).
		Str("hwm_price", currentPrice.String()).
		Msg("TP3 filled: Trailing started")

	return nil
}

// UpdateHWM updates High-Water Mark if current price is higher
func (h *FSMHandler) UpdateHWM(ctx context.Context, positionID uuid.UUID, currentPrice decimal.Decimal) error {
	state, err := h.stateRepo.GetState(ctx, positionID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	// Only update HWM in TRAILING_ACTIVE phase
	if state.Phase != exit.PhaseTrailingActive {
		return nil
	}

	// Check if current price is higher than HWM
	if state.HWMPrice == nil || currentPrice.GreaterThan(*state.HWMPrice) {
		err = h.stateRepo.UpdateHWM(ctx, positionID, currentPrice)
		if err != nil {
			return fmt.Errorf("update hwm: %w", err)
		}

		log.Debug().
			Str("position_id", positionID.String()).
			Str("old_hwm", func() string {
				if state.HWMPrice != nil {
					return state.HWMPrice.String()
				}
				return "nil"
			}()).
			Str("new_hwm", currentPrice.String()).
			Msg("HWM updated")
	}

	return nil
}

// HandleExitFilled handles full exit filled event
func (h *FSMHandler) HandleExitFilled(ctx context.Context, positionID uuid.UUID) error {
	// Update state: any → EXITED
	err := h.stateRepo.UpdatePhase(ctx, positionID, exit.PhaseExited)
	if err != nil {
		return fmt.Errorf("update phase: %w", err)
	}

	log.Info().
		Str("position_id", positionID.String()).
		Str("phase", exit.PhaseExited).
		Msg("Position fully exited")

	return nil
}
