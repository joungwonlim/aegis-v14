import { cn } from "@/lib/utils"

interface ChangeIndicatorProps {
  changePrice: number | undefined
  changeRate: number | undefined
  className?: string
  showPrice?: boolean
  showRate?: boolean
}

/**
 * ChangeIndicator - 전일대비 표시 컴포넌트
 *
 * 상승/하락에 따라 색상과 화살표를 표시합니다.
 * - 상승: 빨간색 ▲
 * - 하락: 파란색 ▼
 * - 변동없음: 회색 -
 *
 * @example
 * <ChangeIndicator changePrice={7} changeRate={0.55} />
 * // 출력: ▲ 7 (+0.55%)
 *
 * <ChangeIndicator changePrice={-1600} changeRate={-0.85} />
 * // 출력: ▼ 1,600 (-0.85%)
 */
export function ChangeIndicator({
  changePrice,
  changeRate,
  className,
  showPrice = true,
  showRate = true,
}: ChangeIndicatorProps) {
  // 데이터가 없는 경우
  if (changePrice === undefined && changeRate === undefined) {
    return <span className={cn("text-muted-foreground", className)}>-</span>
  }

  const price = changePrice ?? 0
  const rate = changeRate ?? 0

  // 상승/하락/변동없음 판단 (changeRate 우선, 없으면 changePrice 기준)
  const changeValue = rate !== 0 ? rate : price
  const isPositive = changeValue > 0
  const isNegative = changeValue < 0
  const isZero = changeValue === 0

  // 색상 결정
  const color = isPositive ? "#EA5455" : isNegative ? "#2196F3" : "#888888"

  // 화살표 결정
  const arrow = isPositive ? "▲" : isNegative ? "▼" : ""

  // 숫자 포맷팅
  const formatNumber = (value: number) => {
    return Math.abs(value).toLocaleString('ko-KR')
  }

  const formatPercent = (value: number) => {
    const sign = value > 0 ? '+' : ''
    return `${sign}${value.toFixed(2)}%`
  }

  // 변동 없음
  if (isZero) {
    return <span className={cn("text-muted-foreground", className)}>-</span>
  }

  return (
    <span className={className} style={{ color }}>
      <span style={{ fontSize: '0.6em' }}>{arrow}</span>{" "}
      {showPrice && <>{formatNumber(price)}</>}
      {showPrice && showRate && " "}
      {showRate && <>({formatPercent(rate)})</>}
    </span>
  )
}
