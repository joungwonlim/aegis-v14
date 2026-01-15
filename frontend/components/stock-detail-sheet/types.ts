/**
 * StockDetailSheet 모듈 타입 정의
 */

/**
 * 종목 기본 정보
 */
export interface StockInfo {
  symbol: string      // 종목코드 (6자리)
  symbolName: string  // 종목명
  market?: string     // KOSPI/KOSDAQ
  sector?: string     // 업종
}

/**
 * 가격 정보
 */
export interface PriceInfo {
  currentPrice: number     // 현재가
  changePrice: number      // 전일대비 (가격)
  changeRate: number       // 전일대비 (%)
  openPrice?: number       // 시가
  highPrice?: number       // 고가
  lowPrice?: number        // 저가
  volume?: number          // 거래량
  value?: number           // 거래대금
  high52w?: number         // 52주 최고가
  low52w?: number          // 52주 최저가
  prevClose?: number       // 전일종가
}

/**
 * 일봉 데이터
 */
export interface DailyPrice {
  date: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

/**
 * StockDetailSheet 탭 종류
 */
export type StockDetailTab = 'price' | 'order' | 'investment' | 'consensus' | 'ai'
