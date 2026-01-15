import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"

interface StockSymbolProps {
  symbol: string
  symbolName?: string
  size?: 'sm' | 'md' | 'lg'
  showCode?: boolean
}

export function StockSymbol({ symbol, symbolName, size = 'md', showCode = true }: StockSymbolProps) {
  // 종목 로고 URL (네이버 금융 API 활용)
  const logoUrl = `https://ssl.pstatic.net/imgstock/fn/real/logo/stock/Stock${symbol}.svg`

  // 종목명이 없으면 종목코드 사용
  const displayName = symbolName || symbol

  // 종목명 첫 글자 추출 (fallback용)
  const fallbackText = displayName.substring(0, 2)

  // 크기 설정
  const sizeClasses = {
    sm: 'h-6 w-6',
    md: 'h-8 w-8',
    lg: 'h-10 w-10',
  }

  const textSizeClasses = {
    sm: 'text-xs',
    md: 'text-sm',
    lg: 'text-base',
  }

  return (
    <div className="flex items-center gap-2">
      <Avatar className={sizeClasses[size]}>
        <AvatarImage src={logoUrl} alt={displayName} />
        <AvatarFallback className="text-xs">{fallbackText}</AvatarFallback>
      </Avatar>
      <div className="flex flex-col">
        <span className={`font-medium ${textSizeClasses[size]}`}>{displayName}</span>
        {showCode && (
          <span className="text-xs text-muted-foreground">{symbol}</span>
        )}
      </div>
    </div>
  )
}
