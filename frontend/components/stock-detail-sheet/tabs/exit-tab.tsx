'use client'

import { useState, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { toast } from 'sonner'
import { setSymbolOverride, deleteSymbolOverride, createExitProfile, type ExitProfile, type CustomExitRule } from '@/lib/api'
import { useExitProfiles, useSymbolOverride } from '@/hooks/useRuntimeData'
import { CustomRulesEditor } from '@/components/CustomRulesEditor'

interface ExitTabProps {
  symbol: string
  symbolName?: string
  holding?: any
  onExitModeToggle?: (enabled: boolean) => void
}

export function ExitTab({
  symbol,
  symbolName,
  holding,
  onExitModeToggle,
}: ExitTabProps) {
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<string>('existing')
  const [selectedProfileId, setSelectedProfileId] = useState<string>('')
  const [reason, setReason] = useState<string>('')

  // Exit Profiles 조회
  const { data: exitProfiles = [] } = useExitProfiles(true)

  // 선택된 Symbol의 Override 조회
  const { data: symbolOverride } = useSymbolOverride(symbol)

  const currentProfileId = symbolOverride?.profile_id

  // 현재 프로필이 있으면 초기값으로 설정
  useEffect(() => {
    if (currentProfileId) {
      setSelectedProfileId(currentProfileId)
    }
  }, [currentProfileId])

  // Set Override Mutation
  const setOverrideMutation = useMutation({
    mutationFn: async () => {
      if (!selectedProfileId) {
        throw new Error('프로필을 선택해주세요')
      }
      if (!reason.trim()) {
        throw new Error('사유를 입력해주세요')
      }
      await setSymbolOverride(symbol, selectedProfileId, reason)
    },
    onSuccess: () => {
      toast.success('Exit 전략이 설정되었습니다', {
        description: `${symbolName || symbol}에 ${exitProfiles.find(p => p.profile_id === selectedProfileId)?.name} 전략이 적용되었습니다.`,
        duration: 10000,
        style: {
          background: '#10b981',
          color: '#ffffff',
          border: '1px solid #059669',
        },
      })
      // Invalidate queries to refetch
      queryClient.invalidateQueries({ queryKey: ['runtime', 'symbol-override', symbol] })
      queryClient.invalidateQueries({ queryKey: ['runtime', 'holdings'] })
      // Reset
      setReason('')
      setSelectedProfileId('')
    },
    onError: (error: Error) => {
      toast.error('설정 실패', {
        description: error.message,
        duration: 10000,
        style: {
          background: '#ef4444',
          color: '#ffffff',
          border: '1px solid #dc2626',
        },
      })
    },
  })

  // Delete Override Mutation
  const deleteOverrideMutation = useMutation({
    mutationFn: async () => {
      await deleteSymbolOverride(symbol)
    },
    onSuccess: () => {
      toast.success('기본 전략으로 복원되었습니다', {
        description: `${symbolName || symbol}이(가) 기본 전략을 사용합니다.`,
        duration: 10000,
        style: {
          background: '#10b981',
          color: '#ffffff',
          border: '1px solid #059669',
        },
      })
      // Invalidate queries
      queryClient.invalidateQueries({ queryKey: ['runtime', 'symbol-override', symbol] })
      queryClient.invalidateQueries({ queryKey: ['runtime', 'holdings'] })
      // Reset
      setReason('')
      setSelectedProfileId('')
    },
    onError: (error: Error) => {
      toast.error('복원 실패', {
        description: error.message,
        duration: 10000,
        style: {
          background: '#ef4444',
          color: '#ffffff',
          border: '1px solid #dc2626',
        },
      })
    },
  })

  // Create Custom Profile Mutation
  const createCustomProfileMutation = useMutation({
    mutationFn: async ({ profileName, rules }: { profileName: string; rules: CustomExitRule[] }) => {
      const profileId = `custom_${symbol}_${Date.now()}`

      // Create profile with custom rules + minimal default config
      await createExitProfile({
        profile_id: profileId,
        name: profileName,
        description: `${symbolName || symbol} 맞춤형 전략`,
        config: {
          atr: {
            ref: 0.02,
            factor_min: 0.7,
            factor_max: 1.6,
          },
          sl1: {
            base_pct: -0.05,
            min_pct: -0.03,
            max_pct: -0.08,
            qty_pct: 0.5,
          },
          sl2: {
            base_pct: -0.10,
            min_pct: -0.08,
            max_pct: -0.15,
            qty_pct: 1.0,
          },
          tp1: {
            base_pct: 0.05,
            min_pct: 0.03,
            max_pct: 0.10,
            qty_pct: 0,
            stop_floor_profit: 0.02,
          },
          tp2: {
            base_pct: 0.10,
            min_pct: 0.08,
            max_pct: 0.15,
            qty_pct: 0,
          },
          tp3: {
            base_pct: 0.15,
            min_pct: 0.12,
            max_pct: 0.20,
            qty_pct: 0,
            start_trailing: false,
          },
          trailing: {
            pct_trail: 0.04,
            atr_k: 2.0,
          },
          time_stop: {
            max_hold_days: 30,
            no_momentum_days: 0,
            no_momentum_profit: 0.02,
          },
          hardstop: {
            enabled: false,
            pct: -0.15,
          },
          custom_rules: rules,
        },
      })

      // Set override to the newly created profile
      await setSymbolOverride(symbol, profileId, `맞춤형 규칙 적용 (${rules.length}개 규칙)`)
    },
    onSuccess: () => {
      toast.success('맞춤형 전략이 생성되었습니다', {
        description: `${symbolName || symbol}에 맞춤형 청산 규칙이 적용되었습니다.`,
        duration: 10000,
        style: {
          background: '#10b981',
          color: '#ffffff',
          border: '1px solid #059669',
        },
      })
      // Invalidate queries
      queryClient.invalidateQueries({ queryKey: ['runtime', 'exit-profiles'] })
      queryClient.invalidateQueries({ queryKey: ['runtime', 'symbol-override', symbol] })
      queryClient.invalidateQueries({ queryKey: ['runtime', 'holdings'] })
      // Reset
      setReason('')
      setSelectedProfileId('')
    },
    onError: (error: Error) => {
      toast.error('생성 실패', {
        description: error.message,
        duration: 10000,
        style: {
          background: '#ef4444',
          color: '#ffffff',
          border: '1px solid #dc2626',
        },
      })
    },
  })

  const handleConfirm = () => {
    setOverrideMutation.mutate()
  }

  const handleRestore = () => {
    deleteOverrideMutation.mutate()
  }

  const handleCustomRulesSave = (profileName: string, rules: CustomExitRule[]) => {
    createCustomProfileMutation.mutate({ profileName, rules })
  }

  const isLoading = setOverrideMutation.isPending || deleteOverrideMutation.isPending || createCustomProfileMutation.isPending

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <div className="text-lg font-semibold">종목별 Exit 전략 설정</div>
        <p className="text-sm text-muted-foreground">
          {symbolName ? `${symbolName} (${symbol})` : symbol}에 적용할 Exit 전략을 선택하세요.
        </p>

        {/* Exit Engine 토글 */}
        {holding && onExitModeToggle && (
          <div className="flex items-center justify-between rounded-lg border bg-muted/50 p-4">
            <div className="space-y-0.5">
              <Label htmlFor="exit-engine-toggle" className="text-base font-semibold">
                Exit Engine
              </Label>
              <div className="text-sm text-muted-foreground">
                자동 손절/익절 시스템 활성화
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Switch
                id="exit-engine-toggle"
                checked={holding.exit_mode === 'ENABLED'}
                onCheckedChange={(enabled) => {
                  onExitModeToggle(enabled)
                }}
              />
              <span className="text-xs text-muted-foreground">
                {holding.exit_mode === 'ENABLED' ? '활성화됨' : '비활성화됨'}
              </span>
            </div>
          </div>
        )}

        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="existing">기존 프로필 선택</TabsTrigger>
            <TabsTrigger value="custom">맞춤 규칙 생성</TabsTrigger>
          </TabsList>

          {/* Tab 1: Existing Profile */}
          <TabsContent value="existing" className="space-y-4">
            <div className="grid gap-4 py-4">
              {/* Profile Selection */}
              <div className="grid gap-2">
                <Label htmlFor="profile">Exit 프로필</Label>
                <Select
                  value={selectedProfileId}
                  onValueChange={setSelectedProfileId}
                  disabled={isLoading}
                >
                  <SelectTrigger id="profile">
                    <SelectValue placeholder="프로필 선택" />
                  </SelectTrigger>
                  <SelectContent>
                    {exitProfiles.map((profile) => (
                      <SelectItem key={profile.profile_id} value={profile.profile_id}>
                        <div className="flex flex-col">
                          <span className="font-medium">{profile.name}</span>
                          {profile.description && (
                            <span className="text-xs text-muted-foreground">{profile.description}</span>
                          )}
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Reason Input */}
              <div className="grid gap-2">
                <Label htmlFor="reason">사유</Label>
                <Input
                  id="reason"
                  placeholder="예: 변동성이 높아 보수적 전략 적용"
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  disabled={isLoading}
                />
              </div>

              {/* Current Override Info */}
              {currentProfileId && (
                <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm">
                  <p className="font-medium text-amber-900">현재 설정</p>
                  <p className="text-amber-700">
                    {exitProfiles.find(p => p.profile_id === currentProfileId)?.name || '알 수 없는 프로필'}
                  </p>
                </div>
              )}
            </div>

            <div className="flex gap-2 justify-end">
              {/* Restore Button (if override exists) */}
              {currentProfileId && (
                <Button
                  variant="outline"
                  onClick={handleRestore}
                  disabled={isLoading}
                >
                  기본 전략으로 복원
                </Button>
              )}

              <Button
                onClick={handleConfirm}
                disabled={isLoading || !selectedProfileId || !reason.trim()}
              >
                {isLoading ? '처리 중...' : '적용'}
              </Button>
            </div>
          </TabsContent>

          {/* Tab 2: Custom Rules */}
          <TabsContent value="custom" className="space-y-4">
            <CustomRulesEditor onSave={handleCustomRulesSave} />
          </TabsContent>
        </Tabs>

        {/* Exit 전략 요약 */}
        <div className="mt-8 space-y-6 border-t pt-6">
          <div className="text-lg font-semibold">Exit 전략 요약</div>

          <div className="grid grid-cols-2 gap-4">
            {/* Strategy For Fall */}
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-blue-500"></div>
                <h3 className="font-semibold text-blue-600 dark:text-blue-400">Strategy For Fall</h3>
              </div>
              <div className="rounded-lg border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-3 py-2 text-left font-medium">트리거</th>
                      <th className="px-3 py-2 text-left font-medium">조건</th>
                      <th className="px-3 py-2 text-left font-medium">액션</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y">
                    <tr className="hover:bg-muted/30">
                      <td className="px-3 py-2 font-medium">HARDSTOP</td>
                      <td className="px-3 py-2 text-muted-foreground">-10%</td>
                      <td className="px-3 py-2 text-blue-600 dark:text-blue-400">전량 청산</td>
                    </tr>
                    <tr className="hover:bg-muted/30">
                      <td className="px-3 py-2 font-medium">SL2</td>
                      <td className="px-3 py-2 text-muted-foreground">-5%</td>
                      <td className="px-3 py-2 text-blue-600 dark:text-blue-400">잔량 100%</td>
                    </tr>
                    <tr className="hover:bg-muted/30">
                      <td className="px-3 py-2 font-medium">SL1</td>
                      <td className="px-3 py-2 text-muted-foreground">-3%</td>
                      <td className="px-3 py-2 text-blue-600 dark:text-blue-400">잔량 50%</td>
                    </tr>
                    <tr className="hover:bg-muted/30">
                      <td className="px-3 py-2 font-medium">Stop Floor</td>
                      <td className="px-3 py-2 text-muted-foreground">본전+0.6%</td>
                      <td className="px-3 py-2 text-blue-600 dark:text-blue-400">잔량 전량</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <p className="text-xs text-muted-foreground px-1">
                💡 Stop Floor는 TP1 체결 후 활성화
              </p>
            </div>

            {/* Strategy For Rise */}
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-red-500"></div>
                <h3 className="font-semibold text-red-600 dark:text-red-400">Strategy For Rise</h3>
              </div>
              <div className="rounded-lg border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-3 py-2 text-left font-medium">트리거</th>
                      <th className="px-3 py-2 text-left font-medium">조건</th>
                      <th className="px-3 py-2 text-left font-medium">액션</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y">
                    <tr className="hover:bg-muted/30">
                      <td className="px-3 py-2 font-medium">TP1</td>
                      <td className="px-3 py-2 text-muted-foreground">+7%</td>
                      <td className="px-3 py-2 text-red-600 dark:text-red-400">원본 10%</td>
                    </tr>
                    <tr className="hover:bg-muted/30">
                      <td className="px-3 py-2 font-medium">TP2</td>
                      <td className="px-3 py-2 text-muted-foreground">+10%</td>
                      <td className="px-3 py-2 text-red-600 dark:text-red-400">원본 20%</td>
                    </tr>
                    <tr className="hover:bg-muted/30">
                      <td className="px-3 py-2 font-medium">TP3</td>
                      <td className="px-3 py-2 text-muted-foreground">+15%</td>
                      <td className="px-3 py-2 text-red-600 dark:text-red-400">원본 30%</td>
                    </tr>
                    <tr className="hover:bg-muted/30">
                      <td className="px-3 py-2 font-medium">Trailing</td>
                      <td className="px-3 py-2 text-muted-foreground">HWM -3%</td>
                      <td className="px-3 py-2 text-red-600 dark:text-red-400">잔량 40%</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <p className="text-xs text-muted-foreground px-1">
                💡 Trailing은 TP3 체결 후 잔량에 적용
              </p>
            </div>
          </div>

          {/* 추가 설명 */}
          <div className="rounded-lg bg-muted/30 p-4 text-sm space-y-2">
            <div className="font-medium">🎯 v14 핵심 특징</div>
            <ul className="space-y-1 text-muted-foreground ml-4">
              <li>• <span className="font-semibold text-foreground">원본 기준 익절</span>: TP1/2/3는 원본 수량 기준으로 계산 (합계 60%)</li>
              <li>• <span className="font-semibold text-foreground">Stop Floor</span>: TP1 체결 즉시 본전+0.6% 보호 활성화</li>
              <li>• <span className="font-semibold text-foreground">Trailing</span>: TP3 이후 잔량 40%는 HWM 대비 -3% 트레일링</li>
              <li>• <span className="font-semibold text-foreground">HARDSTOP</span>: -10% 비상 손절 (PAUSE_ALL 우회)</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  )
}
