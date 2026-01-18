package audit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/audit"
	"github.com/wonny/aegis/v14/internal/infra/kis"
)

// =============================================================================
// KIS Audit Builder
// =============================================================================

// KISAuditBuilder KIS 체결 내역을 audit 데이터로 변환
type KISAuditBuilder struct {
	service            *Service
	kisClient          *kis.Client
	defaultAccountNo   string
	defaultProductCode string
}

// NewKISAuditBuilder 새 KIS Audit Builder 생성
func NewKISAuditBuilder(service *Service, kisClient *kis.Client, defaultAccountNo, defaultProductCode string) *KISAuditBuilder {
	return &KISAuditBuilder{
		service:            service,
		kisClient:          kisClient,
		defaultAccountNo:   defaultAccountNo,
		defaultProductCode: defaultProductCode,
	}
}

// =============================================================================
// Build Audit Data from KIS
// =============================================================================

// BuildFromKIS KIS 체결 내역에서 audit 데이터 생성
func (b *KISAuditBuilder) BuildFromKIS(ctx context.Context, accountNo, accountProductCode string, startDate, endDate time.Time) error {
	// Use defaults if not provided
	if accountNo == "" {
		accountNo = b.defaultAccountNo
	}
	if accountProductCode == "" {
		accountProductCode = b.defaultProductCode
	}

	log.Info().
		Str("account", accountNo).
		Str("product_code", accountProductCode).
		Str("start_date", startDate.Format("2006-01-02")).
		Str("end_date", endDate.Format("2006-01-02")).
		Msg("Building audit data from KIS")

	// 1. KIS에서 체결 내역 조회
	orders, err := b.kisClient.REST.GetFilledOrdersByDateRange(ctx, accountNo, accountProductCode, startDate, endDate)
	if err != nil {
		return fmt.Errorf("get filled orders: %w", err)
	}

	log.Info().Int("orders_count", len(orders)).Msg("Retrieved filled orders from KIS")

	if len(orders) == 0 {
		log.Warn().Msg("No filled orders found in the specified date range")
		return nil
	}

	// 2. 종목별로 그룹화 (매수/매도 매칭용)
	tradesBySymbol := make(map[string][]*audit.Trade)

	for _, order := range orders {
		// Parse order data
		qty, _ := strconv.ParseInt(order.TotalExecQty, 10, 64)
		price, _ := strconv.ParseFloat(order.AvgExecPrice, 64)

		if qty == 0 || price == 0 {
			continue
		}

		// Parse date
		orderDate, err := time.Parse("20060102", order.OrderDate)
		if err != nil {
			log.Warn().Err(err).Str("order_date", order.OrderDate).Msg("Failed to parse order date")
			continue
		}

		// Determine side (01:매도, 02:매수)
		side := "BUY"
		if order.OrderSide == "01" {
			side = "SELL"
		}

		trade := &audit.Trade{
			Symbol:    order.StockCode,
			Side:      side,
			Quantity:  int(qty),
			Price:     int64(price),
			EntryDate: orderDate,
			ExitDate:  orderDate,
		}

		tradesBySymbol[order.StockCode] = append(tradesBySymbol[order.StockCode], trade)
	}

	// 3. 매수/매도 매칭하여 trade_history 생성
	var tradeHistories []audit.Trade
	for symbol, trades := range tradesBySymbol {
		matched := b.matchTrades(symbol, trades)
		tradeHistories = append(tradeHistories, matched...)
	}

	log.Info().Int("trade_histories", len(tradeHistories)).Msg("Matched trades created")

	// 4. trade_history 저장 (아직 DB 메서드 없음 - TODO)
	// TODO: 나중에 trade_history 저장 구현

	// 5. 일별 PnL 계산
	dailyPnLs := b.calculateDailyPnL(tradeHistories, startDate, endDate)

	// 6. daily_pnl 저장
	for _, pnl := range dailyPnLs {
		if err := b.service.RecordDailyPnL(ctx, &pnl); err != nil {
			log.Error().Err(err).Time("date", pnl.Date).Msg("Failed to save daily PnL")
		}
	}

	log.Info().Int("daily_pnls", len(dailyPnLs)).Msg("Daily PnL data saved")

	// 7. 성과 리포트 생성 (1M 기간)
	if time.Since(endDate) < 30*24*time.Hour {
		report, err := b.service.GeneratePerformanceReport(ctx, audit.Period1M)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to generate performance report")
		} else {
			log.Info().Msg("Performance report generated")
			_ = report
		}
	}

	return nil
}

// =============================================================================
// Trade Matching
// =============================================================================

// matchTrades 매수/매도 매칭 (FIFO)
func (b *KISAuditBuilder) matchTrades(symbol string, trades []*audit.Trade) []audit.Trade {
	var buyQueue []*audit.Trade
	var matched []audit.Trade

	// 날짜순 정렬
	// TODO: 더 정교한 정렬 필요

	for _, trade := range trades {
		if trade.Side == "BUY" {
			buyQueue = append(buyQueue, trade)
		} else if trade.Side == "SELL" {
			// 매도 시 FIFO로 매칭
			sellQty := trade.Quantity
			sellPrice := trade.Price

			for sellQty > 0 && len(buyQueue) > 0 {
				buy := buyQueue[0]

				matchQty := buy.Quantity
				if sellQty < matchQty {
					matchQty = sellQty
				}

				// 매칭된 거래 생성
				pnl := float64(matchQty) * (float64(sellPrice) - float64(buy.Price))
				pnlPercent := (float64(sellPrice) - float64(buy.Price)) / float64(buy.Price)
				holdDays := int(trade.ExitDate.Sub(buy.EntryDate).Hours() / 24)

				matched = append(matched, audit.Trade{
					Symbol:     symbol,
					Side:       "SELL",
					Quantity:   matchQty,
					Price:      sellPrice,
					PnL:        pnl,
					PnLPercent: pnlPercent,
					EntryDate:  buy.EntryDate,
					ExitDate:   trade.ExitDate,
					HoldDays:   holdDays,
				})

				// 수량 차감
				buy.Quantity -= matchQty
				sellQty -= matchQty

				// 매수 수량이 0이면 큐에서 제거
				if buy.Quantity == 0 {
					buyQueue = buyQueue[1:]
				}
			}
		}
	}

	return matched
}

// =============================================================================
// Daily PnL Calculation
// =============================================================================

// calculateDailyPnL 일별 손익 계산
func (b *KISAuditBuilder) calculateDailyPnL(trades []audit.Trade, startDate, endDate time.Time) []audit.DailyPnL {
	dailyMap := make(map[time.Time]*audit.DailyPnL)

	// 날짜별로 초기화
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dailyMap[d] = &audit.DailyPnL{
			Date:          d,
			RealizedPnL:   0,
			UnrealizedPnL: 0,
			TotalPnL:      0,
			DailyReturn:   0,
		}
	}

	// 거래별로 실현 손익 집계
	for _, trade := range trades {
		if trade.ExitDate.IsZero() {
			continue
		}

		exitDate := trade.ExitDate.Truncate(24 * time.Hour)
		if pnl, ok := dailyMap[exitDate]; ok {
			pnl.RealizedPnL += int64(trade.PnL)
			pnl.TotalPnL = pnl.RealizedPnL + pnl.UnrealizedPnL
		}
	}

	// map을 slice로 변환
	var result []audit.DailyPnL
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		if pnl, ok := dailyMap[d]; ok {
			result = append(result, *pnl)
		}
	}

	return result
}

// =============================================================================
// Save Trade History (TODO)
// =============================================================================

// SaveTradeHistory trade_history 테이블에 저장
func (b *KISAuditBuilder) SaveTradeHistory(ctx context.Context, trade *audit.Trade) error {
	// TODO: audit repository에 SaveTradeHistory 메서드 추가 필요
	query := `
		INSERT INTO audit.trade_history (
			trade_id, stock_code, stock_name,
			entry_date, entry_price, entry_qty,
			exit_date, exit_price, exit_qty,
			realized_pnl, realized_pnl_pct,
			holding_days
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (trade_id) DO NOTHING
	`

	tradeID := uuid.New()
	_, err := b.service.repo.(interface {
		Exec(ctx context.Context, sql string, arguments ...any) (int64, error)
	}).Exec(ctx, query,
		tradeID,
		trade.Symbol,
		"", // stock_name (TODO: 조회 필요)
		trade.EntryDate,
		trade.Price,
		trade.Quantity,
		trade.ExitDate,
		trade.Price,
		trade.Quantity,
		trade.PnL,
		trade.PnLPercent,
		trade.HoldDays,
	)

	return err
}
