package exit

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/domain/price"
)

// TestEvaluateSL2 tests SL2 trigger evaluation
func TestEvaluateSL2(t *testing.T) {
	svc := &Service{}

	// Setup test data
	snapshot := PositionSnapshot{
		PositionID: uuid.New(),
		Symbol:     "005930",
		Qty:        100,
		AvgPrice:   decimal.NewFromInt(70000),
		Version:    1,
	}

	profile := &exit.ExitProfile{
		Config: exit.ExitProfileConfig{
			SL2: exit.TriggerConfig{
				BasePct: -0.05, // -5%
				QtyPct:  1.00,
			},
		},
	}

	t.Run("SL2 hit", func(t *testing.T) {
		pnlPct := decimal.NewFromFloat(-5.5) // -5.5%

		trigger := svc.evaluateSL2(snapshot, pnlPct, profile)

		if trigger == nil {
			t.Fatal("Expected SL2 trigger, got nil")
		}
		if trigger.ReasonCode != exit.ReasonSL2 {
			t.Errorf("Expected ReasonSL2, got %s", trigger.ReasonCode)
		}
		if trigger.Qty != 100 {
			t.Errorf("Expected qty 100, got %d", trigger.Qty)
		}
		if trigger.OrderType != exit.OrderTypeMKT {
			t.Errorf("Expected MKT order, got %s", trigger.OrderType)
		}
	})

	t.Run("SL2 not hit", func(t *testing.T) {
		pnlPct := decimal.NewFromFloat(-4.0) // -4% (above SL2)

		trigger := svc.evaluateSL2(snapshot, pnlPct, profile)

		if trigger != nil {
			t.Errorf("Expected no trigger, got %+v", trigger)
		}
	})
}

// TestEvaluateSL1 tests SL1 trigger evaluation
func TestEvaluateSL1(t *testing.T) {
	svc := &Service{}

	snapshot := PositionSnapshot{
		PositionID: uuid.New(),
		Symbol:     "005930",
		Qty:        100,
		AvgPrice:   decimal.NewFromInt(70000),
		Version:    1,
	}

	profile := &exit.ExitProfile{
		Config: exit.ExitProfileConfig{
			SL1: exit.TriggerConfig{
				BasePct: -0.03, // -3%
				QtyPct:  0.50,  // 50%
			},
		},
	}

	t.Run("SL1 hit", func(t *testing.T) {
		pnlPct := decimal.NewFromFloat(-3.5) // -3.5%

		trigger := svc.evaluateSL1(snapshot, pnlPct, profile)

		if trigger == nil {
			t.Fatal("Expected SL1 trigger, got nil")
		}
		if trigger.ReasonCode != exit.ReasonSL1 {
			t.Errorf("Expected ReasonSL1, got %s", trigger.ReasonCode)
		}
		if trigger.Qty != 50 {
			t.Errorf("Expected qty 50 (50%% of 100), got %d", trigger.Qty)
		}
		if trigger.OrderType != exit.OrderTypeMKT {
			t.Errorf("Expected MKT order, got %s", trigger.OrderType)
		}
	})

	t.Run("SL1 not hit", func(t *testing.T) {
		pnlPct := decimal.NewFromFloat(-2.0) // -2% (above SL1)

		trigger := svc.evaluateSL1(snapshot, pnlPct, profile)

		if trigger != nil {
			t.Errorf("Expected no trigger, got %+v", trigger)
		}
	})

	t.Run("SL1 qty minimum 1", func(t *testing.T) {
		// Test with 1 share position
		smallSnapshot := PositionSnapshot{
			PositionID: uuid.New(),
			Symbol:     "005930",
			Qty:        1,
			AvgPrice:   decimal.NewFromInt(70000),
			Version:    1,
		}

		pnlPct := decimal.NewFromFloat(-3.5) // -3.5%

		trigger := svc.evaluateSL1(smallSnapshot, pnlPct, profile)

		if trigger == nil {
			t.Fatal("Expected SL1 trigger, got nil")
		}
		if trigger.Qty != 1 {
			t.Errorf("Expected minimum qty 1, got %d", trigger.Qty)
		}
	})
}

// TestEvaluateTP1 tests TP1 trigger evaluation
func TestEvaluateTP1(t *testing.T) {
	svc := &Service{}

	snapshot := PositionSnapshot{
		PositionID: uuid.New(),
		Symbol:     "005930",
		Qty:        100,
		AvgPrice:   decimal.NewFromInt(70000),
		Version:    1,
	}

	profile := &exit.ExitProfile{
		Config: exit.ExitProfileConfig{
			TP1: exit.TriggerConfig{
				BasePct: 0.07, // +7%
				QtyPct:  0.25, // 25%
			},
		},
	}

	t.Run("TP1 hit", func(t *testing.T) {
		pnlPct := decimal.NewFromFloat(7.5) // +7.5%

		trigger := svc.evaluateTP1(snapshot, pnlPct, profile)

		if trigger == nil {
			t.Fatal("Expected TP1 trigger, got nil")
		}
		if trigger.ReasonCode != exit.ReasonTP1 {
			t.Errorf("Expected ReasonTP1, got %s", trigger.ReasonCode)
		}
		if trigger.Qty != 25 {
			t.Errorf("Expected qty 25 (25%% of 100), got %d", trigger.Qty)
		}
		if trigger.OrderType != exit.OrderTypeLMT {
			t.Errorf("Expected LMT order, got %s", trigger.OrderType)
		}
	})

	t.Run("TP1 not hit", func(t *testing.T) {
		pnlPct := decimal.NewFromFloat(5.0) // +5% (below TP1)

		trigger := svc.evaluateTP1(snapshot, pnlPct, profile)

		if trigger != nil {
			t.Errorf("Expected no trigger, got %+v", trigger)
		}
	})
}

// TestEvaluateStopFloor tests Stop Floor trigger evaluation
func TestEvaluateStopFloor(t *testing.T) {
	svc := &Service{}

	snapshot := PositionSnapshot{
		PositionID: uuid.New(),
		Symbol:     "005930",
		Qty:        75, // Remaining after TP1 (25% exit)
		AvgPrice:   decimal.NewFromInt(70000),
		Version:    2,
	}

	stopFloorPrice := decimal.NewFromInt(70420) // 70000 * 1.006

	t.Run("Stop Floor hit", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:          exit.PhaseTP1Done,
			StopFloorPrice: &stopFloorPrice,
		}

		currentPrice := decimal.NewFromInt(70300) // Below Stop Floor

		trigger := svc.evaluateStopFloor(snapshot, currentPrice, state)

		if trigger == nil {
			t.Fatal("Expected Stop Floor trigger, got nil")
		}
		if trigger.ReasonCode != exit.ReasonStopFloor {
			t.Errorf("Expected ReasonStopFloor, got %s", trigger.ReasonCode)
		}
		if trigger.Qty != 75 {
			t.Errorf("Expected full remaining qty 75, got %d", trigger.Qty)
		}
		if trigger.OrderType != exit.OrderTypeMKT {
			t.Errorf("Expected MKT order, got %s", trigger.OrderType)
		}
	})

	t.Run("Stop Floor not hit", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:          exit.PhaseTP1Done,
			StopFloorPrice: &stopFloorPrice,
		}

		currentPrice := decimal.NewFromInt(71000) // Above Stop Floor

		trigger := svc.evaluateStopFloor(snapshot, currentPrice, state)

		if trigger != nil {
			t.Errorf("Expected no trigger, got %+v", trigger)
		}
	})

	t.Run("Stop Floor not set", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:          exit.PhaseTP1Done,
			StopFloorPrice: nil, // Not set
		}

		currentPrice := decimal.NewFromInt(70000)

		trigger := svc.evaluateStopFloor(snapshot, currentPrice, state)

		if trigger != nil {
			t.Errorf("Expected no trigger (Stop Floor not set), got %+v", trigger)
		}
	})
}

// TestEvaluateTrailing tests Trailing Stop trigger evaluation
func TestEvaluateTrailing(t *testing.T) {
	svc := &Service{}

	snapshot := PositionSnapshot{
		PositionID: uuid.New(),
		Symbol:     "005930",
		Qty:        50, // Remaining after TP1+TP2+TP3
		AvgPrice:   decimal.NewFromInt(70000),
		Version:    4,
	}

	profile := &exit.ExitProfile{
		Config: exit.ExitProfileConfig{
			Trailing: exit.TrailingConfig{
				PctTrail: 0.04, // 4%
			},
		},
	}

	hwmPrice := decimal.NewFromInt(85000) // High-Water Mark

	t.Run("Trailing stop hit", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:    exit.PhaseTrailingActive,
			HWMPrice: &hwmPrice,
		}

		// Trailing stop price = 85000 * 0.96 = 81600
		currentPrice := decimal.NewFromInt(81500) // Below trailing stop

		trigger := svc.evaluateTrailing(snapshot, currentPrice, state, profile)

		if trigger == nil {
			t.Fatal("Expected Trailing trigger, got nil")
		}
		if trigger.ReasonCode != exit.ReasonTrail {
			t.Errorf("Expected ReasonTrail, got %s", trigger.ReasonCode)
		}
		if trigger.Qty != 50 {
			t.Errorf("Expected full remaining qty 50, got %d", trigger.Qty)
		}
		if trigger.OrderType != exit.OrderTypeMKT {
			t.Errorf("Expected MKT order, got %s", trigger.OrderType)
		}
	})

	t.Run("Trailing stop not hit", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:    exit.PhaseTrailingActive,
			HWMPrice: &hwmPrice,
		}

		currentPrice := decimal.NewFromInt(82000) // Above trailing stop

		trigger := svc.evaluateTrailing(snapshot, currentPrice, state, profile)

		if trigger != nil {
			t.Errorf("Expected no trigger, got %+v", trigger)
		}
	})

	t.Run("HWM not set", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:    exit.PhaseTrailingActive,
			HWMPrice: nil, // Not set
		}

		currentPrice := decimal.NewFromInt(82000)

		trigger := svc.evaluateTrailing(snapshot, currentPrice, state, profile)

		if trigger != nil {
			t.Errorf("Expected no trigger (HWM not set), got %+v", trigger)
		}
	})
}

// TestTriggerPriority tests trigger priority order
func TestTriggerPriority(t *testing.T) {
	svc := &Service{}

	snapshot := PositionSnapshot{
		PositionID: uuid.New(),
		Symbol:     "005930",
		Qty:        100,
		AvgPrice:   decimal.NewFromInt(70000),
		Version:    1,
	}

	profile := &exit.ExitProfile{
		Config: exit.ExitProfileConfig{
			SL2: exit.TriggerConfig{BasePct: -0.05, QtyPct: 1.00},
			SL1: exit.TriggerConfig{BasePct: -0.03, QtyPct: 0.50},
			TP1: exit.TriggerConfig{BasePct: 0.07, QtyPct: 0.25},
		},
	}

	state := &exit.PositionState{
		Phase: exit.PhaseOpen,
	}

	bestPrice := &price.BestPrice{
		BestPrice: 66000, // -5.7% from 70000
	}

	t.Run("SL2 has highest priority (both SL1 and SL2 triggered)", func(t *testing.T) {
		trigger := svc.evaluateTriggers(snapshot, state, bestPrice, profile, exit.ControlModeRunning)

		if trigger == nil {
			t.Fatal("Expected trigger, got nil")
		}
		if trigger.ReasonCode != exit.ReasonSL2 {
			t.Errorf("Expected SL2 (highest priority), got %s", trigger.ReasonCode)
		}
	})

	t.Run("PAUSE_PROFIT blocks TP triggers", func(t *testing.T) {
		profitBestPrice := &price.BestPrice{
			BestPrice: 75000, // +7.1% (TP1 would trigger)
		}

		trigger := svc.evaluateTriggers(snapshot, state, profitBestPrice, profile, exit.ControlModePauseProfit)

		if trigger != nil {
			t.Errorf("Expected no trigger (TP blocked by PAUSE_PROFIT), got %+v", trigger)
		}
	})

	t.Run("PAUSE_PROFIT allows SL triggers", func(t *testing.T) {
		lossBestPrice := &price.BestPrice{
			BestPrice: 67500, // -3.6% (SL1 triggers)
		}

		trigger := svc.evaluateTriggers(snapshot, state, lossBestPrice, profile, exit.ControlModePauseProfit)

		if trigger == nil {
			t.Fatal("Expected SL1 trigger, got nil")
		}
		if trigger.ReasonCode != exit.ReasonSL1 {
			t.Errorf("Expected SL1, got %s", trigger.ReasonCode)
		}
	})

	t.Run("PAUSE_ALL blocks all triggers", func(t *testing.T) {
		trigger := svc.evaluateTriggers(snapshot, state, bestPrice, profile, exit.ControlModePauseAll)

		if trigger != nil {
			t.Errorf("Expected no trigger (PAUSE_ALL), got %+v", trigger)
		}
	})
}
