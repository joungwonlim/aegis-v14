'use client'

import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical, X } from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { type CustomExitRule } from '@/lib/api'

interface CustomRuleItemProps {
  rule: CustomExitRule
  onUpdate: (updates: Partial<CustomExitRule>) => void
  onDelete: () => void
}

export function CustomRuleItem({ rule, onUpdate, onDelete }: CustomRuleItemProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: rule.id,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex items-center gap-2 rounded-md border bg-card p-3"
    >
      {/* Drag Handle */}
      <div {...attributes} {...listeners} className="cursor-grab">
        <GripVertical className="h-5 w-5 text-muted-foreground" />
      </div>

      {/* Enabled Toggle */}
      <Switch
        checked={rule.enabled}
        onCheckedChange={(enabled) => onUpdate({ enabled })}
      />

      {/* Condition Select */}
      <Select
        value={rule.condition}
        onValueChange={(condition) => onUpdate({ condition: condition as 'profit_above' | 'profit_below' })}
      >
        <SelectTrigger className="w-[140px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="profit_above">수익률 ≥</SelectItem>
          <SelectItem value="profit_below">수익률 ≤</SelectItem>
        </SelectContent>
      </Select>

      {/* Threshold Input */}
      <div className="flex items-center gap-1">
        <Input
          type="number"
          value={rule.threshold}
          onChange={(e) => onUpdate({ threshold: parseFloat(e.target.value) || 0 })}
          className="w-[80px]"
          step="0.1"
        />
        <span className="text-sm text-muted-foreground">%</span>
      </div>

      {/* Arrow */}
      <span className="text-muted-foreground">→</span>

      {/* Exit Percent Input */}
      <div className="flex items-center gap-1">
        <Input
          type="number"
          value={rule.exit_percent}
          onChange={(e) => {
            const value = parseFloat(e.target.value) || 0
            if (value >= 0 && value <= 100) {
              onUpdate({ exit_percent: value })
            }
          }}
          className="w-[80px]"
          step="1"
          min="0"
          max="100"
        />
        <span className="text-sm text-muted-foreground">% 매도</span>
      </div>

      {/* Description Input (Optional) */}
      <Input
        type="text"
        value={rule.description || ''}
        onChange={(e) => onUpdate({ description: e.target.value })}
        placeholder="설명 (선택)"
        className="flex-1 min-w-[120px]"
      />

      {/* Delete Button */}
      <Button
        type="button"
        variant="ghost"
        size="icon"
        onClick={onDelete}
        className="text-destructive hover:text-destructive"
      >
        <X className="h-4 w-4" />
      </Button>
    </div>
  )
}
