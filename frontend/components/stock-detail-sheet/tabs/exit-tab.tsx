'use client'

import { useState, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { toast } from 'sonner'
import { setSymbolOverride, deleteSymbolOverride, createExitProfile, type CustomExitRule } from '@/lib/api'
import { useSymbolOverride, useExitProfiles } from '@/hooks/useRuntimeData'
import { Plus, Trash2, TrendingDown, TrendingUp, Settings2 } from 'lucide-react'

interface ExitTabProps {
  symbol: string
  symbolName?: string
  holding?: any
  onExitModeToggle?: (enabled: boolean) => void
}

interface CustomRule {
  id: string
  condition: 'profit_above' | 'profit_below'
  threshold: number
  exitPercent: number
  enabled: boolean
}

export function ExitTab({
  symbol,
  symbolName,
  holding,
  onExitModeToggle,
}: ExitTabProps) {
  const queryClient = useQueryClient()

  // Strategy 활성화 상태
  const [strategyForFallEnabled, setStrategyForFallEnabled] = useState(true)
  const [strategyForRiseEnabled, setStrategyForRiseEnabled] = useState(true)

  // 맞춤 규칙
  const [customRules, setCustomRules] = useState<CustomRule[]>([])
  const [showCustomRuleForm, setShowCustomRuleForm] = useState(false)
  const [newRule, setNewRule] = useState<Partial<CustomRule>>({
    condition: 'profit_above',
    threshold: 5,
    exitPercent: 10,
  })

  // 선택된 Symbol의 Override 조회
  const { data: symbolOverride } = useSymbolOverride(symbol)

  // Exit 프로필 목록 조회 (저장된 맞춤 규칙 로드용)
  const { data: exitProfiles } = useExitProfiles(false) // activeOnly=false로 모든 프로필 조회

  // 저장된 프로필에서 설정 로드
  useEffect(() => {
    if (symbolOverride?.profile_id && exitProfiles) {
      const profile = exitProfiles.find(p => p.profile_id === symbolOverride.profile_id)
      if (profile?.config) {
        // Helper: 실제 설정값이 있는지 확인 (Go에서 빈 구조체는 base_pct=0으로 반환됨)
        const hasValidTrigger = (trigger?: { base_pct?: number }) => {
          return trigger && trigger.base_pct !== undefined && trigger.base_pct !== 0
        }

        // Strategy For Fall 상태 로드 (sl1, sl2가 유효하거나 hardstop이 enabled인 경우)
        const hasFallStrategy = !!(
          hasValidTrigger(profile.config.sl1) ||
          hasValidTrigger(profile.config.sl2) ||
          profile.config.hardstop?.enabled
        )
        setStrategyForFallEnabled(hasFallStrategy)

        // Strategy For Rise 상태 로드 (tp1, tp2, tp3가 유효하거나 trailing이 있는 경우)
        const hasRiseStrategy = !!(
          hasValidTrigger(profile.config.tp1) ||
          hasValidTrigger(profile.config.tp2) ||
          hasValidTrigger(profile.config.tp3) ||
          (profile.config.trailing && profile.config.trailing.pct_trail !== 0)
        )
        setStrategyForRiseEnabled(hasRiseStrategy)

        // 맞춤 규칙 로드
        if (profile.config.custom_rules && profile.config.custom_rules.length > 0) {
          // API 형식의 custom_rules를 UI 형식으로 변환
          // Backend stores percentage values (4.0 = 4%, -4.0 = -4%)
          const loadedRules: CustomRule[] = profile.config.custom_rules.map(r => ({
            id: r.id,
            condition: r.condition,
            threshold: Math.abs(r.threshold), // 4 or -4 → 4
            exitPercent: r.exit_percent, // Already in percentage format
            enabled: r.enabled,
          }))
          setCustomRules(loadedRules)
        } else {
          setCustomRules([])
        }
      }
    }
  }, [symbolOverride?.profile_id, exitProfiles])

  // Create Custom Profile Mutation
  const createCustomProfileMutation = useMutation({
    mutationFn: async () => {
      const profileId = `custom_${symbol}_${Date.now()}`

      // Build custom rules from UI state
      // Note: Backend expects percentage values (4.0 = 4%), NOT decimals (0.04)
      const rules: CustomExitRule[] = customRules
        .filter(r => r.enabled)
        .map((r, idx) => ({
          id: r.id,
          enabled: r.enabled,
          condition: r.condition,
          threshold: r.condition === 'profit_above' ? r.threshold : -r.threshold, // profit_above: +4, profit_below: -4
          exit_percent: r.exitPercent,
          priority: idx + 1,
          description: `${r.condition === 'profit_above' ? '+' : '-'}${Math.abs(r.threshold)}%/${r.exitPercent}% ${r.condition === 'profit_above' ? '익절' : '손절'}`,
        }))

      await createExitProfile({
        profile_id: profileId,
        name: `${symbolName || symbol} 맞춤 전략`,
        description: `맞춤형 청산 규칙 (${rules.length}개)`,
        config: {
          // Strategy For Fall (손절)
          sl1: strategyForFallEnabled ? {
            base_pct: -0.03,
            min_pct: -0.02,
            max_pct: -0.05,
            qty_pct: 0.5,
          } : undefined,
          sl2: strategyForFallEnabled ? {
            base_pct: -0.05,
            min_pct: -0.03,
            max_pct: -0.08,
            qty_pct: 1.0,
          } : undefined,
          hardstop: strategyForFallEnabled ? {
            enabled: true,
            pct: -0.07,
          } : { enabled: false, pct: -0.07 },

          // Strategy For Rise (익절)
          tp1: strategyForRiseEnabled ? {
            base_pct: 0.07,
            min_pct: 0.05,
            max_pct: 0.10,
            qty_pct: 0.10,
            stop_floor_profit: 0.006,
          } : undefined,
          tp2: strategyForRiseEnabled ? {
            base_pct: 0.10,
            min_pct: 0.08,
            max_pct: 0.15,
            qty_pct: 0.20,
          } : undefined,
          tp3: strategyForRiseEnabled ? {
            base_pct: 0.15,
            min_pct: 0.12,
            max_pct: 0.20,
            qty_pct: 0.30,
            start_trailing: true,
          } : undefined,
          trailing: strategyForRiseEnabled ? {
            pct_trail: 0.03,
            atr_k: 2.0,
          } : undefined,

          // 맞춤 규칙
          custom_rules: rules.length > 0 ? rules : undefined,
        },
      })

      await setSymbolOverride(symbol, profileId, '맞춤형 Exit 전략 적용')
    },
    onSuccess: () => {
      toast.success('Exit 전략이 저장되었습니다', {
        description: `${symbolName || symbol}에 맞춤형 전략이 적용되었습니다.`,
      })
      queryClient.invalidateQueries({ queryKey: ['runtime', 'exit-profiles'] })
      queryClient.invalidateQueries({ queryKey: ['runtime', 'symbol-override', symbol] })
      queryClient.invalidateQueries({ queryKey: ['runtime', 'holdings'] })
    },
    onError: (error: Error) => {
      toast.error('저장 실패', { description: error.message })
    },
  })

  // 맞춤 규칙 추가
  const addCustomRule = () => {
    if (!newRule.threshold || !newRule.exitPercent) return

    const rule: CustomRule = {
      id: `rule_${Date.now()}`,
      condition: newRule.condition || 'profit_above',
      threshold: newRule.threshold,
      exitPercent: newRule.exitPercent,
      enabled: true,
    }

    setCustomRules([...customRules, rule])
    setNewRule({ condition: 'profit_above', threshold: 5, exitPercent: 10 })
    setShowCustomRuleForm(false)
  }

  // 맞춤 규칙 삭제
  const removeCustomRule = (id: string) => {
    setCustomRules(customRules.filter(r => r.id !== id))
  }

  // 맞춤 규칙 토글
  const toggleCustomRule = (id: string) => {
    setCustomRules(customRules.map(r =>
      r.id === id ? { ...r, enabled: !r.enabled } : r
    ))
  }

  const handleSave = () => {
    createCustomProfileMutation.mutate()
  }

  const isLoading = createCustomProfileMutation.isPending

  return (
    <div className="space-y-6">
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

      {/* Strategy For Fall (손절 전략) */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <TrendingDown className="h-5 w-5 text-blue-500" />
            <h3 className="text-base font-semibold">Strategy For Fall</h3>
            <Badge variant={strategyForFallEnabled ? "default" : "secondary"} className="text-xs">
              {strategyForFallEnabled ? "적용" : "미적용"}
            </Badge>
          </div>
          <div className="flex items-center gap-2">
            <Checkbox
              id="strategy-fall"
              checked={strategyForFallEnabled}
              onCheckedChange={(checked) => setStrategyForFallEnabled(checked === true)}
            />
            <Label htmlFor="strategy-fall" className="text-sm cursor-pointer">적용</Label>
          </div>
        </div>

        <div className={`rounded-lg border transition-opacity ${!strategyForFallEnabled ? 'opacity-50' : ''}`}>
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
                <td className="px-3 py-2 text-muted-foreground">-7% 이하</td>
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

      {/* Strategy For Rise (익절 전략) */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <TrendingUp className="h-5 w-5 text-red-500" />
            <h3 className="text-base font-semibold">Strategy For Rise</h3>
            <Badge variant={strategyForRiseEnabled ? "destructive" : "secondary"} className="text-xs">
              {strategyForRiseEnabled ? "적용" : "미적용"}
            </Badge>
          </div>
          <div className="flex items-center gap-2">
            <Checkbox
              id="strategy-rise"
              checked={strategyForRiseEnabled}
              onCheckedChange={(checked) => setStrategyForRiseEnabled(checked === true)}
            />
            <Label htmlFor="strategy-rise" className="text-sm cursor-pointer">적용</Label>
          </div>
        </div>

        <div className={`rounded-lg border transition-opacity ${!strategyForRiseEnabled ? 'opacity-50' : ''}`}>
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
                <td className="px-3 py-2 text-red-600 dark:text-red-400">잔량 10%</td>
              </tr>
              <tr className="hover:bg-muted/30">
                <td className="px-3 py-2 font-medium">TP2</td>
                <td className="px-3 py-2 text-muted-foreground">+10%</td>
                <td className="px-3 py-2 text-red-600 dark:text-red-400">잔량 20%</td>
              </tr>
              <tr className="hover:bg-muted/30">
                <td className="px-3 py-2 font-medium">TP3</td>
                <td className="px-3 py-2 text-muted-foreground">+15%</td>
                <td className="px-3 py-2 text-red-600 dark:text-red-400">잔량 30%</td>
              </tr>
              <tr className="hover:bg-muted/30 bg-amber-50/50 dark:bg-amber-950/20">
                <td className="px-3 py-2 font-medium">부분 Trail</td>
                <td className="px-3 py-2 text-muted-foreground">TP2 후 HWM -3%</td>
                <td className="px-3 py-2 text-red-600 dark:text-red-400">잔량 50%</td>
              </tr>
              <tr className="hover:bg-muted/30 bg-amber-50/50 dark:bg-amber-950/20">
                <td className="px-3 py-2 font-medium">전량 Trail</td>
                <td className="px-3 py-2 text-muted-foreground">TP3 후 HWM -3%</td>
                <td className="px-3 py-2 text-red-600 dark:text-red-400">잔량 50%</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div className="text-xs text-muted-foreground px-1 space-y-1">
          <p>💡 <span className="font-medium">TP1</span>: Stop Floor 활성화 (본전+0.6% 방어)</p>
          <p>💡 <span className="font-medium">TP2 후</span>: HWM -3% 도달 시 잔량 50% 부분 익절</p>
          <p>💡 <span className="font-medium">TP3 후</span>: HWM -3% 도달 시 잔량 50% 청산</p>
        </div>
      </div>

      {/* 맞춤 규칙 생성 */}
      <div className="space-y-3 border-t pt-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Settings2 className="h-5 w-5 text-purple-500" />
            <h3 className="text-base font-semibold">맞춤 규칙</h3>
            {customRules.length > 0 && (
              <Badge variant="outline" className="text-xs">
                {customRules.filter(r => r.enabled).length}개 활성
              </Badge>
            )}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowCustomRuleForm(!showCustomRuleForm)}
          >
            <Plus className="h-4 w-4 mr-1" />
            규칙 추가
          </Button>
        </div>

        {/* 맞춤 규칙 추가 폼 */}
        {showCustomRuleForm && (
          <div className="rounded-lg border bg-muted/30 p-4 space-y-3">
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1">
                <Label className="text-xs">조건</Label>
                <select
                  className="w-full h-9 px-3 rounded-md border bg-background text-sm"
                  value={newRule.condition}
                  onChange={(e) => setNewRule({ ...newRule, condition: e.target.value as 'profit_above' | 'profit_below' })}
                >
                  <option value="profit_above">수익률 이상</option>
                  <option value="profit_below">손실률 이하</option>
                </select>
              </div>
              <div className="space-y-1">
                <Label className="text-xs">임계값 (%)</Label>
                <Input
                  type="number"
                  placeholder="예: 5"
                  value={newRule.threshold || ''}
                  onChange={(e) => setNewRule({ ...newRule, threshold: Number(e.target.value) })}
                  className="h-9"
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">청산 비율 (%)</Label>
                <Input
                  type="number"
                  placeholder="예: 10"
                  value={newRule.exitPercent || ''}
                  onChange={(e) => setNewRule({ ...newRule, exitPercent: Number(e.target.value) })}
                  className="h-9"
                />
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" size="sm" onClick={() => setShowCustomRuleForm(false)}>
                취소
              </Button>
              <Button size="sm" onClick={addCustomRule}>
                추가
              </Button>
            </div>
          </div>
        )}

        {/* 맞춤 규칙 목록 */}
        {customRules.length > 0 && (
          <div className="rounded-lg border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="px-3 py-2 text-left font-medium w-10">활성</th>
                  <th className="px-3 py-2 text-left font-medium">조건</th>
                  <th className="px-3 py-2 text-left font-medium">임계값</th>
                  <th className="px-3 py-2 text-left font-medium">청산비율</th>
                  <th className="px-3 py-2 text-left font-medium w-10"></th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {customRules.map((rule) => (
                  <tr key={rule.id} className={`hover:bg-muted/30 ${!rule.enabled ? 'opacity-50' : ''}`}>
                    <td className="px-3 py-2">
                      <Checkbox
                        checked={rule.enabled}
                        onCheckedChange={() => toggleCustomRule(rule.id)}
                      />
                    </td>
                    <td className="px-3 py-2">
                      {rule.condition === 'profit_above' ? '수익률 ≥' : '손실률 ≤'}
                    </td>
                    <td className="px-3 py-2 font-medium">
                      {rule.condition === 'profit_above' ? '+' : '-'}{Math.abs(rule.threshold)}%
                    </td>
                    <td className="px-3 py-2 text-purple-600 dark:text-purple-400">
                      잔량 {rule.exitPercent}%
                    </td>
                    <td className="px-3 py-2">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7"
                        onClick={() => removeCustomRule(rule.id)}
                      >
                        <Trash2 className="h-4 w-4 text-muted-foreground hover:text-destructive" />
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {customRules.length === 0 && !showCustomRuleForm && (
          <div className="rounded-lg border border-dashed p-4 text-center text-sm text-muted-foreground">
            맞춤 규칙이 없습니다. 상단의 "규칙 추가" 버튼을 클릭하여 추가하세요.
          </div>
        )}
      </div>

      {/* 저장 버튼 */}
      <div className="flex justify-end pt-4 border-t">
        <Button onClick={handleSave} disabled={isLoading}>
          {isLoading ? '저장 중...' : '전략 저장'}
        </Button>
      </div>
    </div>
  )
}
