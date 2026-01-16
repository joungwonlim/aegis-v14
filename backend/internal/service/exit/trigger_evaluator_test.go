package exit

import (
	"testing"
	"time"

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
		EntryTS:    time.Now().Add(-24 * time.Hour), // 1 day ago
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

		trigger := svc.evaluateSL2(snapshot, pnlPct, profile, 1.0)

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

		trigger := svc.evaluateSL2(snapshot, pnlPct, profile, 1.0)

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
		EntryTS:    time.Now().Add(-24 * time.Hour), // 1 day ago
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

		trigger := svc.evaluateSL1(snapshot, pnlPct, profile, 1.0)

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

		trigger := svc.evaluateSL1(snapshot, pnlPct, profile, 1.0)

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
			EntryTS:    time.Now().Add(-24 * time.Hour),
			Version:    1,
		}

		pnlPct := decimal.NewFromFloat(-3.5) // -3.5%

		trigger := svc.evaluateSL1(smallSnapshot, pnlPct, profile, 1.0)

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
		EntryTS:    time.Now().Add(-24 * time.Hour), // 1 day ago
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

		trigger := svc.evaluateTP1(snapshot, pnlPct, profile, 1.0)

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

		trigger := svc.evaluateTP1(snapshot, pnlPct, profile, 1.0)

		if trigger != nil {
			t.Errorf("Expected no trigger, got %+v", trigger)
		}
	})
}

// TestEvaluateTP2 tests TP2 trigger evaluation
func TestEvaluateTP2(t *testing.T) {
	svc := &Service{}

	snapshot := PositionSnapshot{
		PositionID: uuid.New(),
		Symbol:     "005930",
		Qty:        100,
		AvgPrice:   decimal.NewFromInt(70000),
		EntryTS:    time.Now().Add(-24 * time.Hour), // 1 day ago
		Version:    1,
	}

	profile := &exit.ExitProfile{
		Config: exit.ExitProfileConfig{
			TP2: exit.TriggerConfig{
				BasePct: 0.10, // +10%
				QtyPct:  0.25, // 25%
			},
		},
	}

	t.Run("TP2 hit", func(t *testing.T) {
		pnlPct := decimal.NewFromFloat(10.5) // +10.5%

		trigger := svc.evaluateTP2(snapshot, pnlPct, profile, 1.0)

		if trigger == nil {
			t.Fatal("Expected TP2 trigger, got nil")
		}
		if trigger.ReasonCode != exit.ReasonTP2 {
			t.Errorf("Expected ReasonTP2, got %s", trigger.ReasonCode)
		}
		if trigger.Qty != 25 {
			t.Errorf("Expected qty 25 (25%% of 100), got %d", trigger.Qty)
		}
		if trigger.OrderType != exit.OrderTypeLMT {
			t.Errorf("Expected LMT order, got %s", trigger.OrderType)
		}
	})

	t.Run("TP2 not hit", func(t *testing.T) {
		pnlPct := decimal.NewFromFloat(8.0) // +8% (below TP2)

		trigger := svc.evaluateTP2(snapshot, pnlPct, profile, 1.0)

		if trigger != nil {
			t.Errorf("Expected no trigger, got %+v", trigger)
		}
	})
}

// TestEvaluateTP3 tests TP3 trigger evaluation
func TestEvaluateTP3(t *testing.T) {
	svc := &Service{}

	snapshot := PositionSnapshot{
		PositionID: uuid.New(),
		Symbol:     "005930",
		Qty:        100,
		AvgPrice:   decimal.NewFromInt(70000),
		EntryTS:    time.Now().Add(-24 * time.Hour), // 1 day ago
		Version:    1,
	}

	profile := &exit.ExitProfile{
		Config: exit.ExitProfileConfig{
			TP3: exit.TriggerConfig{
				BasePct: 0.16, // +16%
				QtyPct:  0.20, // 20%
			},
		},
	}

	t.Run("TP3 hit", func(t *testing.T) {
		pnlPct := decimal.NewFromFloat(17.0) // +17%

		trigger := svc.evaluateTP3(snapshot, pnlPct, profile, 1.0)

		if trigger == nil {
			t.Fatal("Expected TP3 trigger, got nil")
		}
		if trigger.ReasonCode != exit.ReasonTP3 {
			t.Errorf("Expected ReasonTP3, got %s", trigger.ReasonCode)
		}
		if trigger.Qty != 20 {
			t.Errorf("Expected qty 20 (20%% of 100), got %d", trigger.Qty)
		}
		if trigger.OrderType != exit.OrderTypeLMT {
			t.Errorf("Expected LMT order, got %s", trigger.OrderType)
		}
	})

	t.Run("TP3 not hit", func(t *testing.T) {
		pnlPct := decimal.NewFromFloat(12.0) // +12% (below TP3)

		trigger := svc.evaluateTP3(snapshot, pnlPct, profile, 1.0)

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
		EntryTS:    time.Now().Add(-24 * time.Hour),
		Version:    2,
	}

	stopFloorPrice := decimal.NewFromInt(70420) // 70000 * 1.006

	t.Run("Stop Floor - first breach (no trigger yet)", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:                exit.PhaseTP1Done,
			StopFloorPrice:       &stopFloorPrice,
			StopFloorBreachTicks: 0, // First breach
		}

		currentPrice := decimal.NewFromInt(70300) // Below Stop Floor

		trigger := svc.evaluateStopFloor(snapshot, currentPrice, state)

		if trigger != nil {
			t.Errorf("Expected no trigger on first breach, got %+v", trigger)
		}
	})

	t.Run("Stop Floor - second breach but not confirmed (tick=1)", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:                exit.PhaseTP1Done,
			StopFloorPrice:       &stopFloorPrice,
			StopFloorBreachTicks: 1, // Second breach, not yet >= 2
		}

		currentPrice := decimal.NewFromInt(70300) // Below Stop Floor

		trigger := svc.evaluateStopFloor(snapshot, currentPrice, state)

		if trigger != nil {
			t.Errorf("Expected no trigger when StopFloorBreachTicks=1, got %+v", trigger)
		}
	})

	t.Run("Stop Floor - confirmed breach (tick=2, triggers)", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:                exit.PhaseTP1Done,
			StopFloorPrice:       &stopFloorPrice,
			StopFloorBreachTicks: 2, // Confirmed (2 consecutive breaches)
		}

		currentPrice := decimal.NewFromInt(70300) // Below Stop Floor

		trigger := svc.evaluateStopFloor(snapshot, currentPrice, state)

		if trigger == nil {
			t.Fatal("Expected Stop Floor trigger when StopFloorBreachTicks >= 2, got nil")
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
		EntryTS:    time.Now().Add(-24 * time.Hour),
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

	t.Run("Trailing stop - first breach (no trigger yet)", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:               exit.PhaseTrailingActive,
			HWMPrice:            &hwmPrice,
			TrailingBreachTicks: 0, // First breach
		}

		// Trailing stop price = 85000 * 0.96 = 81600
		currentPrice := decimal.NewFromInt(81500) // Below trailing stop

		trigger := svc.evaluateTrailing(snapshot, currentPrice, state, profile)

		if trigger != nil {
			t.Errorf("Expected no trigger on first breach, got %+v", trigger)
		}
	})

	t.Run("Trailing stop - second breach but not confirmed (tick=1)", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:               exit.PhaseTrailingActive,
			HWMPrice:            &hwmPrice,
			TrailingBreachTicks: 1, // Second breach, not yet >= 2
		}

		// Trailing stop price = 85000 * 0.96 = 81600
		currentPrice := decimal.NewFromInt(81500) // Below trailing stop

		trigger := svc.evaluateTrailing(snapshot, currentPrice, state, profile)

		if trigger != nil {
			t.Errorf("Expected no trigger when TrailingBreachTicks=1, got %+v", trigger)
		}
	})

	t.Run("Trailing stop - confirmed breach (tick=2, triggers)", func(t *testing.T) {
		state := &exit.PositionState{
			Phase:               exit.PhaseTrailingActive,
			HWMPrice:            &hwmPrice,
			TrailingBreachTicks: 2, // Confirmed (2 consecutive breaches)
		}

		// Trailing stop price = 85000 * 0.96 = 81600
		currentPrice := decimal.NewFromInt(81500) // Below trailing stop

		trigger := svc.evaluateTrailing(snapshot, currentPrice, state, profile)

		if trigger == nil {
			t.Fatal("Expected Trailing trigger when TrailingBreachTicks >= 2, got nil")
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
		EntryTS:    time.Now().Add(-24 * time.Hour), // 1 day ago
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

// TestATRScaling tests ATR dynamic scaling
func TestATRScaling(t *testing.T) {
	svc := &Service{}

	snapshot := PositionSnapshot{
		PositionID: uuid.New(),
		Symbol:     "005930",
		Qty:        100,
		AvgPrice:   decimal.NewFromInt(70000),
		EntryTS:    time.Now().Add(-24 * time.Hour),
		Version:    1,
	}

	profile := &exit.ExitProfile{
		Config: exit.ExitProfileConfig{
			SL2: exit.TriggerConfig{
				BasePct: -0.05, // -5%
				MinPct:  -0.03, // -3% (min loss, tighter stop)
				MaxPct:  -0.08, // -8% (max loss, wider stop)
				QtyPct:  1.00,
			},
			TP1: exit.TriggerConfig{
				BasePct: 0.07, // +7%
				MinPct:  0.05, // +5% (min gain, tighter target)
				MaxPct:  0.10, // +10% (max gain, wider target)
				QtyPct:  0.25,
			},
		},
	}

	t.Run("ATR factor 1.0 (normal volatility)", func(t *testing.T) {
		// ATR factor = 1.0 (no scaling)
		// SL2 threshold = -5% * 1.0 = -5%
		pnlPct := decimal.NewFromFloat(-5.1) // -5.1% (trigger)

		trigger := svc.evaluateSL2(snapshot, pnlPct, profile, 1.0)

		if trigger == nil {
			t.Fatal("Expected SL2 trigger, got nil")
		}
		if trigger.ReasonCode != exit.ReasonSL2 {
			t.Errorf("Expected ReasonSL2, got %s", trigger.ReasonCode)
		}
	})

	t.Run("ATR factor 1.5 (high volatility - wider stop)", func(t *testing.T) {
		// ATR factor = 1.5 (high volatility)
		// SL2 threshold = -5% * 1.5 = -7.5%
		pnlPct := decimal.NewFromFloat(-6.0) // -6% (no trigger, wider stop)

		trigger := svc.evaluateSL2(snapshot, pnlPct, profile, 1.5)

		if trigger != nil {
			t.Errorf("Expected no trigger (wider stop due to high volatility), got %+v", trigger)
		}

		// But -7.6% should trigger
		pnlPct = decimal.NewFromFloat(-7.6)
		trigger = svc.evaluateSL2(snapshot, pnlPct, profile, 1.5)

		if trigger == nil {
			t.Fatal("Expected SL2 trigger at -7.6%, got nil")
		}
	})

	t.Run("ATR factor 0.7 (low volatility - tighter stop)", func(t *testing.T) {
		// ATR factor = 0.7 (low volatility)
		// SL2 threshold = -5% * 0.7 = -3.5%
		// -3.5% is within bounds (MinPct=-3%, MaxPct=-8%), no clamping
		// -3.6% triggers (less than or equal to -3.5%)
		pnlPct := decimal.NewFromFloat(-3.6) // -3.6% (trigger)

		trigger := svc.evaluateSL2(snapshot, pnlPct, profile, 0.7)

		if trigger == nil {
			t.Fatal("Expected SL2 trigger, got nil")
		}

		// But -3.4% should not trigger (greater than -3.5%)
		pnlPct = decimal.NewFromFloat(-3.4)
		trigger = svc.evaluateSL2(snapshot, pnlPct, profile, 0.7)

		if trigger != nil {
			t.Errorf("Expected no trigger at -3.4%% (not enough loss), got %+v", trigger)
		}
	})

	t.Run("TP1 with high volatility (wider target)", func(t *testing.T) {
		// ATR factor = 1.4 (high volatility)
		// TP1 threshold = +7% * 1.4 = +9.8%
		pnlPct := decimal.NewFromFloat(8.0) // +8% (no trigger, wider target)

		trigger := svc.evaluateTP1(snapshot, pnlPct, profile, 1.4)

		if trigger != nil {
			t.Errorf("Expected no trigger (wider target due to high volatility), got %+v", trigger)
		}

		// But +10% should trigger
		pnlPct = decimal.NewFromFloat(10.0)
		trigger = svc.evaluateTP1(snapshot, pnlPct, profile, 1.4)

		if trigger == nil {
			t.Fatal("Expected TP1 trigger at +10%, got nil")
		}
	})

	t.Run("TP1 clamped to MaxPct", func(t *testing.T) {
		// ATR factor = 2.0 (extremely high volatility)
		// TP1 threshold = +7% * 2.0 = +14%
		// But clamped to MaxPct = +10%
		pnlPct := decimal.NewFromFloat(10.1) // +10.1% (trigger at max bound)

		trigger := svc.evaluateTP1(snapshot, pnlPct, profile, 2.0)

		if trigger == nil {
			t.Fatal("Expected TP1 trigger (clamped to MaxPct), got nil")
		}
	})
}

// TestEvaluateTimeStop tests TIME_STOP trigger evaluation
func TestEvaluateTimeStop(t *testing.T) {
	svc := &Service{}

	currentPrice := decimal.NewFromInt(72000) // +2.86% from 70000

	t.Run("Max hold days exceeded", func(t *testing.T) {
		snapshot := PositionSnapshot{
			PositionID: uuid.New(),
			Symbol:     "005930",
			Qty:        100,
			AvgPrice:   decimal.NewFromInt(70000),
			EntryTS:    time.Now().Add(-31 * 24 * time.Hour), // 31 days ago
			Version:    1,
		}

		profile := &exit.ExitProfile{
			Config: exit.ExitProfileConfig{
				TimeStop: exit.TimeStopConfig{
					MaxHoldDays: 30,
				},
			},
		}

		state := &exit.PositionState{
			Phase: exit.PhaseOpen,
		}

		trigger := svc.evaluateTimeStop(snapshot, state, currentPrice, profile)

		if trigger == nil {
			t.Fatal("Expected TIME_STOP trigger (max hold days), got nil")
		}
		if trigger.ReasonCode != exit.ReasonTime {
			t.Errorf("Expected ReasonTime, got %s", trigger.ReasonCode)
		}
		if trigger.Qty != 100 {
			t.Errorf("Expected full qty 100, got %d", trigger.Qty)
		}
		if trigger.OrderType != exit.OrderTypeMKT {
			t.Errorf("Expected MKT order, got %s", trigger.OrderType)
		}
	})

	t.Run("Max hold days not exceeded", func(t *testing.T) {
		snapshot := PositionSnapshot{
			PositionID: uuid.New(),
			Symbol:     "005930",
			Qty:        100,
			AvgPrice:   decimal.NewFromInt(70000),
			EntryTS:    time.Now().Add(-20 * 24 * time.Hour), // 20 days ago
			Version:    1,
		}

		profile := &exit.ExitProfile{
			Config: exit.ExitProfileConfig{
				TimeStop: exit.TimeStopConfig{
					MaxHoldDays: 30,
				},
			},
		}

		state := &exit.PositionState{
			Phase: exit.PhaseOpen,
		}

		trigger := svc.evaluateTimeStop(snapshot, state, currentPrice, profile)

		if trigger != nil {
			t.Errorf("Expected no trigger (within max hold days), got %+v", trigger)
		}
	})

	t.Run("No momentum with HWM", func(t *testing.T) {
		snapshot := PositionSnapshot{
			PositionID: uuid.New(),
			Symbol:     "005930",
			Qty:        100,
			AvgPrice:   decimal.NewFromInt(70000),
			EntryTS:    time.Now().Add(-11 * 24 * time.Hour), // 11 days ago
			Version:    1,
		}

		hwmPrice := decimal.NewFromInt(70280) // Only +0.4% max profit

		profile := &exit.ExitProfile{
			Config: exit.ExitProfileConfig{
				TimeStop: exit.TimeStopConfig{
					MaxHoldDays:      30,
					NoMomentumDays:   10,
					NoMomentumProfit: 0.01, // Require 1% min profit
				},
			},
		}

		state := &exit.PositionState{
			Phase:    exit.PhaseTrailingActive,
			HWMPrice: &hwmPrice,
		}

		trigger := svc.evaluateTimeStop(snapshot, state, currentPrice, profile)

		if trigger == nil {
			t.Fatal("Expected TIME_STOP trigger (no momentum with HWM), got nil")
		}
		if trigger.ReasonCode != exit.ReasonTime {
			t.Errorf("Expected ReasonTime, got %s", trigger.ReasonCode)
		}
	})

	t.Run("No momentum without HWM (current price)", func(t *testing.T) {
		snapshot := PositionSnapshot{
			PositionID: uuid.New(),
			Symbol:     "005930",
			Qty:        100,
			AvgPrice:   decimal.NewFromInt(70000),
			EntryTS:    time.Now().Add(-11 * 24 * time.Hour), // 11 days ago
			Version:    1,
		}

		lowCurrentPrice := decimal.NewFromInt(70280) // Only +0.4% current profit

		profile := &exit.ExitProfile{
			Config: exit.ExitProfileConfig{
				TimeStop: exit.TimeStopConfig{
					MaxHoldDays:      30,
					NoMomentumDays:   10,
					NoMomentumProfit: 0.01, // Require 1% min profit
				},
			},
		}

		state := &exit.PositionState{
			Phase:    exit.PhaseOpen,
			HWMPrice: nil, // No HWM, use current price
		}

		trigger := svc.evaluateTimeStop(snapshot, state, lowCurrentPrice, profile)

		if trigger == nil {
			t.Fatal("Expected TIME_STOP trigger (no momentum, no HWM), got nil")
		}
		if trigger.ReasonCode != exit.ReasonTime {
			t.Errorf("Expected ReasonTime, got %s", trigger.ReasonCode)
		}
	})

	t.Run("No momentum not triggered (sufficient profit)", func(t *testing.T) {
		snapshot := PositionSnapshot{
			PositionID: uuid.New(),
			Symbol:     "005930",
			Qty:        100,
			AvgPrice:   decimal.NewFromInt(70000),
			EntryTS:    time.Now().Add(-11 * 24 * time.Hour), // 11 days ago
			Version:    1,
		}

		hwmPrice := decimal.NewFromInt(71000) // +1.43% max profit (above threshold)

		profile := &exit.ExitProfile{
			Config: exit.ExitProfileConfig{
				TimeStop: exit.TimeStopConfig{
					MaxHoldDays:      30,
					NoMomentumDays:   10,
					NoMomentumProfit: 0.01, // Require 1% min profit
				},
			},
		}

		state := &exit.PositionState{
			Phase:    exit.PhaseTrailingActive,
			HWMPrice: &hwmPrice,
		}

		trigger := svc.evaluateTimeStop(snapshot, state, currentPrice, profile)

		if trigger != nil {
			t.Errorf("Expected no trigger (sufficient profit), got %+v", trigger)
		}
	})

	t.Run("No momentum not triggered (insufficient days)", func(t *testing.T) {
		snapshot := PositionSnapshot{
			PositionID: uuid.New(),
			Symbol:     "005930",
			Qty:        100,
			AvgPrice:   decimal.NewFromInt(70000),
			EntryTS:    time.Now().Add(-8 * 24 * time.Hour), // Only 8 days ago
			Version:    1,
		}

		hwmPrice := decimal.NewFromInt(70280) // Only +0.4% max profit

		profile := &exit.ExitProfile{
			Config: exit.ExitProfileConfig{
				TimeStop: exit.TimeStopConfig{
					MaxHoldDays:      30,
					NoMomentumDays:   10, // Need 10 days
					NoMomentumProfit: 0.01,
				},
			},
		}

		state := &exit.PositionState{
			Phase:    exit.PhaseTrailingActive,
			HWMPrice: &hwmPrice,
		}

		trigger := svc.evaluateTimeStop(snapshot, state, currentPrice, profile)

		if trigger != nil {
			t.Errorf("Expected no trigger (insufficient days), got %+v", trigger)
		}
	})

	t.Run("MaxHoldDays disabled (0)", func(t *testing.T) {
		snapshot := PositionSnapshot{
			PositionID: uuid.New(),
			Symbol:     "005930",
			Qty:        100,
			AvgPrice:   decimal.NewFromInt(70000),
			EntryTS:    time.Now().Add(-100 * 24 * time.Hour), // 100 days ago
			Version:    1,
		}

		profile := &exit.ExitProfile{
			Config: exit.ExitProfileConfig{
				TimeStop: exit.TimeStopConfig{
					MaxHoldDays: 0, // Disabled
				},
			},
		}

		state := &exit.PositionState{
			Phase: exit.PhaseOpen,
		}

		trigger := svc.evaluateTimeStop(snapshot, state, currentPrice, profile)

		if trigger != nil {
			t.Errorf("Expected no trigger (MaxHoldDays disabled), got %+v", trigger)
		}
	})
}
