package kis

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// ExecutionAdapter adapts KIS Client to execution.KISAdapter interface
type ExecutionAdapter struct {
	client *Client
}

// NewExecutionAdapter creates a new ExecutionAdapter
func NewExecutionAdapter(client *Client) *ExecutionAdapter {
	return &ExecutionAdapter{
		client: client,
	}
}

// SubmitOrder submits an order to KIS
func (a *ExecutionAdapter) SubmitOrder(ctx context.Context, req execution.KISOrderRequest) (*execution.KISOrderResponse, error) {
	// TODO: Implement actual KIS order submission
	// For now, return a placeholder response
	return &execution.KISOrderResponse{
		OrderID:   fmt.Sprintf("KIS_%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Raw:       make(map[string]any),
	}, nil
}

// GetUnfilledOrders retrieves unfilled orders from KIS
func (a *ExecutionAdapter) GetUnfilledOrders(ctx context.Context, accountID string) ([]*execution.KISUnfilledOrder, error) {
	// TODO: Implement actual KIS unfilled orders retrieval
	return []*execution.KISUnfilledOrder{}, nil
}

// GetFills retrieves fills since timestamp
func (a *ExecutionAdapter) GetFills(ctx context.Context, accountID string, since time.Time) ([]*execution.KISFill, error) {
	// TODO: Implement actual KIS fills retrieval
	return []*execution.KISFill{}, nil
}

// GetFillsForOrder retrieves fills for a specific order
func (a *ExecutionAdapter) GetFillsForOrder(ctx context.Context, orderID string) ([]*execution.KISFill, error) {
	// TODO: Implement actual KIS fills retrieval for specific order
	return []*execution.KISFill{}, nil
}

// GetHoldings retrieves holdings from KIS
func (a *ExecutionAdapter) GetHoldings(ctx context.Context, accountID string) ([]*execution.KISHolding, error) {
	// Parse account ID (format: XXXXXXXX-XX)
	parts := strings.Split(accountID, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid account ID format: %s (expected: XXXXXXXX-XX)", accountID)
	}
	accountNo := parts[0]
	accountProductCode := parts[1]

	// Call KIS API
	holdings, err := a.client.REST.GetHoldings(ctx, accountNo, accountProductCode)
	if err != nil {
		return nil, fmt.Errorf("get holdings from KIS: %w", err)
	}

	// Convert to KISHolding
	result := make([]*execution.KISHolding, 0, len(holdings))
	for _, h := range holdings {
		// Parse qty
		qty, err := strconv.ParseInt(h.HoldingQty, 10, 64)
		if err != nil {
			continue // Skip invalid holdings
		}
		if qty <= 0 {
			continue // Skip zero holdings
		}

		// Parse avg price
		avgPrice, err := decimal.NewFromString(h.AvgPurchasePrice)
		if err != nil {
			continue
		}

		// Parse current price
		currentPrice, err := decimal.NewFromString(h.CurrentPrice)
		if err != nil {
			continue
		}

		// Parse PnL
		pnl, err := decimal.NewFromString(h.EvaluateProfitLoss)
		if err != nil {
			pnl = decimal.Zero
		}

		// Parse PnL %
		pnlPct, err := strconv.ParseFloat(h.EvaluateProfitLossRate, 64)
		if err != nil {
			pnlPct = 0.0
		}

		result = append(result, &execution.KISHolding{
			AccountID:    accountID,
			Symbol:       h.Symbol,
			Qty:          qty,
			AvgPrice:     avgPrice,
			CurrentPrice: currentPrice,
			Pnl:          pnl,
			PnlPct:       pnlPct,
			Raw: map[string]any{
				"symbol_name":           h.SymbolName,
				"evaluate_amount":       h.EvaluateAmount,
				"purchase_amount":       h.PurchaseAmount,
				"evaluate_profit_loss":  h.EvaluateProfitLoss,
				"evaluate_profit_loss_rate": h.EvaluateProfitLossRate,
			},
		})
	}

	return result, nil
}

// Ensure ExecutionAdapter implements execution.KISAdapter
var _ execution.KISAdapter = (*ExecutionAdapter)(nil)
