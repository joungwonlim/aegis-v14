'use client'

import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Activity, Database, Clock, CheckCircle2, XCircle, Loader2 } from 'lucide-react'

// API 함수 (나중에 lib/api.ts로 이동)
async function getFetcherStatus() {
  // Placeholder - 나중에 실제 API로 교체
  return {
    active: true,
    lastRun: new Date().toISOString(),
    nextRun: new Date(Date.now() + 3600000).toISOString(),
  }
}

async function getTableStats() {
  // Placeholder - 나중에 실제 API로 교체
  return {
    tables: [
      {
        name: 'stocks',
        displayName: '종목 마스터',
        count: 2547,
        lastUpdate: new Date(Date.now() - 3600000).toISOString(),
        status: 'active',
      },
      {
        name: 'daily_prices',
        displayName: '일봉 데이터',
        count: 1234567,
        lastUpdate: new Date(Date.now() - 7200000).toISOString(),
        status: 'active',
      },
      {
        name: 'investor_flow',
        displayName: '투자자별 수급',
        count: 987654,
        lastUpdate: new Date(Date.now() - 10800000).toISOString(),
        status: 'active',
      },
      {
        name: 'fundamentals',
        displayName: '재무 데이터',
        count: 54321,
        lastUpdate: new Date(Date.now() - 86400000).toISOString(),
        status: 'stale',
      },
      {
        name: 'market_cap',
        displayName: '시가총액',
        count: 43210,
        lastUpdate: new Date(Date.now() - 14400000).toISOString(),
        status: 'active',
      },
      {
        name: 'disclosures',
        displayName: 'DART 공시',
        count: 12345,
        lastUpdate: new Date(Date.now() - 21600000).toISOString(),
        status: 'active',
      },
    ],
  }
}

export default function FetcherPage() {
  const { data: fetcherStatus, isLoading: statusLoading } = useQuery({
    queryKey: ['fetcherStatus'],
    queryFn: getFetcherStatus,
    refetchInterval: 30000, // 30초마다 갱신
  })

  const { data: tableStats, isLoading: statsLoading } = useQuery({
    queryKey: ['tableStats'],
    queryFn: getTableStats,
    refetchInterval: 60000, // 1분마다 갱신
  })

  const formatTimestamp = (ts: string) => {
    return new Date(ts).toLocaleString('ko-KR', {
      timeZone: 'Asia/Seoul',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }

  const formatRelativeTime = (ts: string) => {
    const now = Date.now()
    const then = new Date(ts).getTime()
    const diff = Math.floor((now - then) / 1000) // seconds

    if (diff < 60) return `${diff}초 전`
    if (diff < 3600) return `${Math.floor(diff / 60)}분 전`
    if (diff < 86400) return `${Math.floor(diff / 3600)}시간 전`
    return `${Math.floor(diff / 86400)}일 전`
  }

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Fetcher</h1>
        <p className="text-muted-foreground mt-1">
          데이터 수집 시스템 모니터링
        </p>
      </div>

      {/* Fetcher 상태 */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Activity className="h-5 w-5" />
                Fetcher 상태
              </CardTitle>
              <CardDescription>실시간 데이터 수집 시스템 상태</CardDescription>
            </div>
            {statusLoading ? (
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            ) : (
              <Badge variant={fetcherStatus?.active ? 'default' : 'destructive'}>
                {fetcherStatus?.active ? (
                  <><CheckCircle2 className="h-3 w-3 mr-1" /> 활성화</>
                ) : (
                  <><XCircle className="h-3 w-3 mr-1" /> 비활성화</>
                )}
              </Badge>
            )}
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <div className="text-muted-foreground">마지막 실행</div>
              <div className="font-medium font-mono">
                {fetcherStatus?.lastRun ? formatTimestamp(fetcherStatus.lastRun) : '-'}
              </div>
              <div className="text-xs text-muted-foreground">
                {fetcherStatus?.lastRun ? formatRelativeTime(fetcherStatus.lastRun) : ''}
              </div>
            </div>
            <div>
              <div className="text-muted-foreground">다음 실행 예정</div>
              <div className="font-medium font-mono">
                {fetcherStatus?.nextRun ? formatTimestamp(fetcherStatus.nextRun) : '-'}
              </div>
              <div className="text-xs text-muted-foreground">
                {fetcherStatus?.nextRun && new Date(fetcherStatus.nextRun).getTime() > Date.now()
                  ? `${Math.floor((new Date(fetcherStatus.nextRun).getTime() - Date.now()) / 60000)}분 후`
                  : ''}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 데이터베이스 테이블 통계 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            데이터베이스 테이블
          </CardTitle>
          <CardDescription>data 스키마 테이블별 레코드 수 및 업데이트 상태</CardDescription>
        </CardHeader>
        <CardContent>
          {statsLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>테이블</TableHead>
                  <TableHead className="text-right">레코드 수</TableHead>
                  <TableHead className="text-right">최근 업데이트</TableHead>
                  <TableHead className="text-center">상태</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tableStats?.tables.map((table) => (
                  <TableRow key={table.name}>
                    <TableCell>
                      <div className="font-medium">{table.displayName}</div>
                      <div className="text-xs text-muted-foreground font-mono">
                        data.{table.name}
                      </div>
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {table.count.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="text-sm">{formatRelativeTime(table.lastUpdate)}</div>
                      <div className="text-xs text-muted-foreground font-mono">
                        {formatTimestamp(table.lastUpdate)}
                      </div>
                    </TableCell>
                    <TableCell className="text-center">
                      <Badge
                        variant={table.status === 'active' ? 'default' : 'secondary'}
                        className="text-xs"
                      >
                        {table.status === 'active' ? (
                          <><CheckCircle2 className="h-3 w-3 mr-1" /> 정상</>
                        ) : (
                          <><Clock className="h-3 w-3 mr-1" /> 오래됨</>
                        )}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* 스케줄 정보 (Placeholder) */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Clock className="h-5 w-5" />
            스케줄 정보
          </CardTitle>
          <CardDescription>데이터 수집 스케줄 현황 (구현 예정)</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-sm text-muted-foreground">
            스케줄 상태 표시 기능은 추후 구현 예정입니다.
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
