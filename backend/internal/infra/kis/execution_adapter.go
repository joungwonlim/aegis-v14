package kis

import (
	"context"
	"fmt"
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
	// Parse account ID (format: XXXXXXXX-XX)
	parts := strings.Split(req.AccountID, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid account ID format: %s (expected: XXXXXXXX-XX)", req.AccountID)
	}
	accountNo := parts[0]
	accountProductCode := parts[1]

	// Convert order type
	orderType := "limit"
	if req.OrderType == "MKT" || req.OrderType == "market" {
		orderType = "market"
	}

	// Convert side
	side := "buy"
	if req.Side == execution.SideSell || req.Side == "sell" || req.Side == "SELL" {
		side = "sell"
	}

	// Get price
	var price int64 = 0
	if req.LimitPrice != nil {
		price = req.LimitPrice.IntPart()
	}

	// Call KIS API
	result, err := a.client.REST.PlaceOrder(ctx, accountNo, accountProductCode, req.Symbol, side, orderType, req.Qty, price)
	if err != nil {
		return nil, fmt.Errorf("place order to KIS: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("KIS order failed: %s", result.Message)
	}

	return &execution.KISOrderResponse{
		OrderID:   result.OrderNo,
		Timestamp: time.Now(),
		Raw: map[string]any{
			"message":    result.Message,
			"order_time": result.OrderTime,
		},
	}, nil
}

// CancelOrder cancels an order in KIS
func (a *ExecutionAdapter) CancelOrder(ctx context.Context, accountID string, orderNo string) (*execution.KISCancelResponse, error) {
	// Parse account ID (format: XXXXXXXX-XX)
	parts := strings.Split(accountID, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid account ID format: %s (expected: XXXXXXXX-XX)", accountID)
	}
	accountNo := parts[0]
	accountProductCode := parts[1]

	// Call KIS API
	result, err := a.client.REST.CancelOrder(ctx, accountNo, accountProductCode, orderNo)
	if err != nil {
		return nil, fmt.Errorf("cancel order from KIS: %w", err)
	}

	return &execution.KISCancelResponse{
		OrderNo:   orderNo,
		CancelNo:  result.OrderNo,
		Timestamp: time.Now(),
		Raw: map[string]any{
			"message":    result.Message,
			"order_time": result.OrderTime,
		},
	}, nil
}

// GetUnfilledOrders retrieves unfilled orders from KIS
func (a *ExecutionAdapter) GetUnfilledOrders(ctx context.Context, accountID string) ([]*execution.KISUnfilledOrder, error) {
	// Parse account ID (format: XXXXXXXX-XX)
	parts := strings.Split(accountID, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid account ID format: %s (expected: XXXXXXXX-XX)", accountID)
	}
	accountNo := parts[0]
	accountProductCode := parts[1]

	// Call KIS API
	orders, err := a.client.REST.GetUnfilledOrders(ctx, accountNo, accountProductCode)
	if err != nil {
		return nil, fmt.Errorf("get unfilled orders from KIS: %w", err)
	}

	// Convert to KISUnfilledOrder
	result := make([]*execution.KISUnfilledOrder, 0, len(orders))
	for _, o := range orders {
		orderQty, _ := strconv.ParseInt(o.OrderQty, 10, 64)
		filledQty, _ := strconv.ParseInt(o.TotalExecQty, 10, 64)
		remainingQty, _ := strconv.ParseInt(o.RemainingQty, 10, 64)

		// 미체결 수량이 0이면 스킵
		if remainingQty <= 0 {
			continue
		}

		// 주문상태 결정
		status := "pending"
		if filledQty > 0 {
			status = "partial"
		}
		if o.CancelYN == "Y" {
			status = "cancelled"
		}

		result = append(result, &execution.KISUnfilledOrder{
			OrderID:   o.OrderNo,
			Symbol:    o.StockCode,
			Qty:       orderQty,
			OpenQty:   remainingQty,
			FilledQty: filledQty,
			Status:    status,
			Raw: map[string]any{
				"stock_name":     o.StockName,
				"order_side":     o.OrderSideName,
				"order_price":    o.OrderPrice,
				"avg_exec_price": o.AvgExecPrice,
				"order_time":     o.OrderTime,
			},
		})
	}

	return result, nil
}

// GetFills retrieves fills since timestamp
func (a *ExecutionAdapter) GetFills(ctx context.Context, accountID string, since time.Time) ([]*execution.KISFill, error) {
	// Parse account ID (format: XXXXXXXX-XX)
	parts := strings.Split(accountID, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid account ID format: %s (expected: XXXXXXXX-XX)", accountID)
	}
	accountNo := parts[0]
	accountProductCode := parts[1]

	// Call KIS API (체결된 주문 조회)
	orders, err := a.client.REST.GetFilledOrders(ctx, accountNo, accountProductCode)
	if err != nil {
		return nil, fmt.Errorf("get filled orders from KIS: %w", err)
	}

	// Convert to KISFill
	result := make([]*execution.KISFill, 0, len(orders))
	for i, o := range orders {
		filledQty, _ := strconv.ParseInt(o.TotalExecQty, 10, 64)
		if filledQty <= 0 {
			continue
		}

		avgPrice, _ := decimal.NewFromString(o.AvgExecPrice)
		totalAmount, _ := decimal.NewFromString(o.TotalExecAmount)

		result = append(result, &execution.KISFill{
			ExecID:    fmt.Sprintf("%s-%s", o.OrderNo, o.OrderTime),
			OrderID:   o.OrderNo,
			Symbol:    o.StockCode,
			Qty:       filledQty,
			Price:     avgPrice,
			Fee:       decimal.Zero, // KIS API에서 별도 제공 안함
			Tax:       decimal.Zero,
			Timestamp: time.Now(),
			Seq:       i,
			Raw: map[string]any{
				"stock_name":       o.StockName,
				"order_side":       o.OrderSideName,
				"order_qty":        o.OrderQty,
				"order_price":      o.OrderPrice,
				"exec_amount":      totalAmount.String(),
				"cancel_yn":        o.CancelYN,
				"order_type":       o.OrderTypeName,
			},
		})
	}

	return result, nil
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

		// Calculate actual avg price (including fees) = purchase_amount / qty
		// This matches HTS display (pchs_amt / hldg_qty)
		purchaseAmount, err := decimal.NewFromString(h.PurchaseAmount)
		if err != nil {
			continue
		}
		avgPrice := purchaseAmount.Div(decimal.NewFromInt(qty))

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
				"market":                h.GetMarket(), // KOSPI/KOSDAQ/UNKNOWN
			},
		})
	}

	return result, nil
}

// Ensure ExecutionAdapter implements execution.KISAdapter
var _ execution.KISAdapter = (*ExecutionAdapter)(nil)
