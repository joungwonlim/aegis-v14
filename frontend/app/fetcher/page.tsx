'use client'

import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Activity, Database, Clock, CheckCircle2, XCircle, Loader2 } from 'lucide-react'
import { getTableStats, getFetchLogs } from '@/lib/api'

interface FetcherStatus {
  active: boolean
  lastRun: string
  nextRun: string
}

export default function FetcherPage() {
  // Fetcher status will be implemented later
  const fetcherStatus = null as FetcherStatus | null
  const statusLoading = false

  const { data: tableStatsData, isLoading: statsLoading } = useQuery({
    queryKey: ['tableStats'],
    queryFn: getTableStats,
    refetchInterval: 60000, // 1분마다 갱신
  })

  const { data: fetchLogsData, isLoading: logsLoading } = useQuery({
    queryKey: ['fetchLogs'],
    queryFn: getFetchLogs,
    refetchInterval: 60000, // 1분마다 갱신
  })

  const tableStats = tableStatsData?.tables || []
  const fetchLogs = fetchLogsData?.logs || []

  // Create a map of table name to latest fetch log
  const fetchLogMap = new Map(
    fetchLogs.map(log => [log.target_table, log])
  )

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
            ) : fetcherStatus ? (
              <Badge variant={fetcherStatus.active ? 'default' : 'destructive'}>
                {fetcherStatus.active ? (
                  <><CheckCircle2 className="h-3 w-3 mr-1" /> 활성화</>
                ) : (
                  <><XCircle className="h-3 w-3 mr-1" /> 비활성화</>
                )}
              </Badge>
            ) : (
              <Badge variant="secondary">
                <Clock className="h-3 w-3 mr-1" /> 구현 예정
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
                  <TableHead className="text-right">현재 레코드 수</TableHead>
                  <TableHead className="text-right">마지막 실행</TableHead>
                  <TableHead className="text-right">최근 업데이트</TableHead>
                  <TableHead className="text-center">상태</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tableStats.map((table) => {
                  const log = fetchLogMap.get(table.name)
                  return (
                    <TableRow key={table.name}>
                      <TableCell>
                        <div className="font-medium">{table.display_name}</div>
                        <div className="text-xs text-muted-foreground font-mono">
                          data.{table.name}
                        </div>
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {table.count.toLocaleString()}
                      </TableCell>
                      <TableCell className="text-right">
                        {log ? (
                          <>
                            <div className="text-sm font-medium">
                              {log.records_fetched.toLocaleString()}건 조회
                            </div>
                            <div className="text-xs text-muted-foreground">
                              {log.records_inserted > 0 && `추가: ${log.records_inserted.toLocaleString()} `}
                              {log.records_updated > 0 && `갱신: ${log.records_updated.toLocaleString()}`}
                            </div>
                            <div className="text-xs text-muted-foreground font-mono">
                              {formatRelativeTime(log.started_at)}
                            </div>
                          </>
                        ) : (
                          <div className="text-sm text-muted-foreground">-</div>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        {table.last_update ? (
                          <>
                            <div className="text-sm">{formatRelativeTime(table.last_update)}</div>
                            <div className="text-xs text-muted-foreground font-mono">
                              {formatTimestamp(table.last_update)}
                            </div>
                          </>
                        ) : (
                          <div className="text-sm text-muted-foreground">-</div>
                        )}
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
                  )
                })}
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
