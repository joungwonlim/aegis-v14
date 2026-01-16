'use client'

import { useState, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import { CustomRulesEditor } from './CustomRulesEditor'

interface SymbolOverrideDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  symbol: string
  symbolName?: string
  currentProfileId?: string | null
  profiles: ExitProfile[]
  holding?: any // Exit Mode 확인 및 변경용
  onExitModeToggle?: (holding: any, enabled: boolean) => void
}

export function SymbolOverrideDialog({
  open,
  onOpenChange,
  symbol,
  symbolName,
  currentProfileId,
  profiles,
  holding,
  onExitModeToggle,
}: SymbolOverrideDialogProps) {
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<string>('existing')
  const [selectedProfileId, setSelectedProfileId] = useState<string>('')
  const [reason, setReason] = useState<string>('')

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
        description: `${symbolName || symbol}에 ${profiles.find(p => p.profile_id === selectedProfileId)?.name} 전략이 적용되었습니다.`,
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
      // Reset and close
      setReason('')
      setSelectedProfileId('')
      onOpenChange(false)
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
      // Reset and close
      setReason('')
      setSelectedProfileId('')
      onOpenChange(false)
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
      // Reset and close
      setReason('')
      setSelectedProfileId('')
      onOpenChange(false)
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>종목별 Exit 전략 설정</DialogTitle>
          <DialogDescription>
            {symbolName ? `${symbolName} (${symbol})` : symbol}에 적용할 Exit 전략을 선택하세요.
          </DialogDescription>
        </DialogHeader>

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
                onCheckedChange={(enabled) => onExitModeToggle(holding, enabled)}
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
                    {profiles.map((profile) => (
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
                    {profiles.find(p => p.profile_id === currentProfileId)?.name || '알 수 없는 프로필'}
                  </p>
                </div>
              )}
            </div>

            <DialogFooter className="gap-2">
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
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={isLoading}
              >
                취소
              </Button>

              <Button
                onClick={handleConfirm}
                disabled={isLoading || !selectedProfileId || !reason.trim()}
              >
                {isLoading ? '처리 중...' : '적용'}
              </Button>
            </DialogFooter>
          </TabsContent>

          {/* Tab 2: Custom Rules */}
          <TabsContent value="custom" className="space-y-4">
            <CustomRulesEditor onSave={handleCustomRulesSave} />
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  )
}
