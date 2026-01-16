'use client'

import { useState } from 'react'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { CustomRuleItem } from './CustomRuleItem'
import { type CustomExitRule } from '@/lib/api'

interface CustomRulesEditorProps {
  onSave: (profileName: string, rules: CustomExitRule[]) => void
}

export function CustomRulesEditor({ onSave }: CustomRulesEditorProps) {
  const [profileName, setProfileName] = useState('')
  const [rules, setRules] = useState<CustomExitRule[]>([])

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event

    if (over && active.id !== over.id) {
      setRules((items) => {
        const oldIndex = items.findIndex((item) => item.id === active.id)
        const newIndex = items.findIndex((item) => item.id === over.id)

        const newItems = arrayMove(items, oldIndex, newIndex)

        // Update priorities based on new order
        return newItems.map((item, index) => ({
          ...item,
          priority: index,
        }))
      })
    }
  }

  const addRule = () => {
    const newRule: CustomExitRule = {
      id: crypto.randomUUID(),
      enabled: true,
      condition: 'profit_above',
      threshold: 7,
      exit_percent: 20,
      priority: rules.length,
      description: '',
    }
    setRules([...rules, newRule])
  }

  const updateRule = (id: string, updates: Partial<CustomExitRule>) => {
    setRules((items) =>
      items.map((item) => (item.id === id ? { ...item, ...updates } : item))
    )
  }

  const deleteRule = (id: string) => {
    setRules((items) => {
      const filtered = items.filter((item) => item.id !== id)
      // Reassign priorities
      return filtered.map((item, index) => ({
        ...item,
        priority: index,
      }))
    })
  }

  const handleSave = () => {
    if (!profileName.trim()) {
      alert('프로필 이름을 입력해주세요')
      return
    }
    if (rules.length === 0) {
      alert('최소 1개의 규칙을 추가해주세요')
      return
    }
    onSave(profileName, rules)
  }

  return (
    <div className="space-y-4">
      {/* Profile Name Input */}
      <div className="grid gap-2">
        <Label htmlFor="profile-name">프로필 이름</Label>
        <Input
          id="profile-name"
          placeholder="예: 삼성전자 맞춤형 전략"
          value={profileName}
          onChange={(e) => setProfileName(e.target.value)}
        />
      </div>

      {/* Rules List */}
      <div className="grid gap-2">
        <div className="flex items-center justify-between">
          <Label>청산 규칙</Label>
          <Button type="button" variant="outline" size="sm" onClick={addRule}>
            + 규칙 추가
          </Button>
        </div>

        {rules.length === 0 ? (
          <div className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
            규칙을 추가하여 맞춤형 청산 전략을 만드세요
          </div>
        ) : (
          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext items={rules.map((r) => r.id)} strategy={verticalListSortingStrategy}>
              <div className="space-y-2">
                {rules.map((rule) => (
                  <CustomRuleItem
                    key={rule.id}
                    rule={rule}
                    onUpdate={(updates) => updateRule(rule.id, updates)}
                    onDelete={() => deleteRule(rule.id)}
                  />
                ))}
              </div>
            </SortableContext>
          </DndContext>
        )}
      </div>

      {/* Info Box */}
      <div className="rounded-md border border-blue-200 bg-blue-50 p-3 text-sm">
        <p className="font-medium text-blue-900">규칙 평가 순서</p>
        <p className="text-blue-700">
          규칙은 위에서 아래로 순서대로 평가되며, 한 번 실행된 규칙은 재실행되지 않습니다.
        </p>
      </div>

      {/* Save Button */}
      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={!profileName.trim() || rules.length === 0}>
          저장
        </Button>
      </div>
    </div>
  )
}
