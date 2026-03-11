import * as React from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Eye, EyeOff, Plus, Trash2, Save, X } from 'lucide-react'

interface VariableTableProps {
  /** Variables as key→value map. Values can be string, number, boolean, etc. */
  variables?: Record<string, unknown>
  /** Secrets as key→masked-value map. */
  secrets?: Record<string, string>
  /** Called when user saves changes. Receives updated variables and secrets. */
  onSave?: (variables: Record<string, unknown>, secrets: Record<string, string>) => void
  /** Whether save is in progress. */
  isSaving?: boolean
}

interface EditableRow {
  key: string
  value: string
  isSecret: boolean
  isNew?: boolean
}

/**
 * Editable variable table with secrets masked by default.
 * Supports add/update/delete with a save-all action.
 */
export function VariableTable({ variables = {}, secrets = {}, onSave, isSaving }: VariableTableProps) {
  const [isEditing, setIsEditing] = React.useState(false)
  const [revealedSecrets, setRevealedSecrets] = React.useState<Set<string>>(new Set())
  const [editRows, setEditRows] = React.useState<EditableRow[]>([])

  // Build rows from variables + secrets
  const readOnlyRows = React.useMemo(() => {
    const rows: { key: string; value: string; isSecret: boolean }[] = []
    for (const [k, v] of Object.entries(variables)) {
      rows.push({ key: k, value: String(v), isSecret: false })
    }
    for (const [k, v] of Object.entries(secrets)) {
      rows.push({ key: k, value: v, isSecret: true })
    }
    return rows.sort((a, b) => a.key.localeCompare(b.key))
  }, [variables, secrets])

  const startEditing = () => {
    setEditRows(readOnlyRows.map((r) => ({ ...r })))
    setIsEditing(true)
  }

  const cancelEditing = () => {
    setIsEditing(false)
    setEditRows([])
  }

  const addRow = () => {
    setEditRows([...editRows, { key: '', value: '', isSecret: false, isNew: true }])
  }

  const updateRow = (index: number, field: 'key' | 'value', val: string) => {
    setEditRows((prev) => prev.map((r, i) => (i === index ? { ...r, [field]: val } : r)))
  }

  const toggleSecret = (index: number) => {
    setEditRows((prev) =>
      prev.map((r, i) => (i === index ? { ...r, isSecret: !r.isSecret } : r)),
    )
  }

  const removeRow = (index: number) => {
    setEditRows((prev) => prev.filter((_, i) => i !== index))
  }

  const handleSave = () => {
    const newVars: Record<string, unknown> = {}
    const newSecrets: Record<string, string> = {}

    for (const row of editRows) {
      if (!row.key.trim()) continue
      if (row.isSecret) {
        newSecrets[row.key.trim()] = row.value
      } else {
        newVars[row.key.trim()] = row.value
      }
    }

    onSave?.(newVars, newSecrets)
    setIsEditing(false)
  }

  const toggleReveal = (key: string) => {
    setRevealedSecrets((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  const isEmpty = readOnlyRows.length === 0 && !isEditing

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-muted-foreground">Variables & Secrets</h3>
        <div className="flex items-center gap-2">
          {isEditing ? (
            <>
              <Button variant="ghost" size="sm" onClick={cancelEditing} disabled={isSaving}>
                <X className="h-3.5 w-3.5 mr-1" />
                Cancel
              </Button>
              <Button size="sm" onClick={handleSave} disabled={isSaving}>
                <Save className="h-3.5 w-3.5 mr-1" />
                {isSaving ? 'Saving…' : 'Save'}
              </Button>
            </>
          ) : (
            <Button variant="outline" size="sm" onClick={startEditing}>
              Edit
            </Button>
          )}
        </div>
      </div>

      {isEmpty ? (
        <div className="text-center py-8 text-muted-foreground text-sm">
          No variables defined. Click <strong>Edit</strong> to add some.
        </div>
      ) : isEditing ? (
        <div className="space-y-2">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[200px]">Key</TableHead>
                <TableHead>Value</TableHead>
                <TableHead className="w-[80px]">Type</TableHead>
                <TableHead className="w-[50px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {editRows.map((row, idx) => (
                <TableRow key={idx}>
                  <TableCell>
                    <Input
                      value={row.key}
                      onChange={(e) => updateRow(idx, 'key', e.target.value)}
                      placeholder="KEY"
                      className="h-8 text-xs font-mono"
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      value={row.value}
                      onChange={(e) => updateRow(idx, 'value', e.target.value)}
                      placeholder="value"
                      type={row.isSecret ? 'password' : 'text'}
                      className="h-8 text-xs font-mono"
                    />
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => toggleSecret(idx)}
                      className="h-7 text-xs"
                    >
                      {row.isSecret ? (
                        <Badge variant="destructive" className="text-[10px]">secret</Badge>
                      ) : (
                        <Badge variant="secondary" className="text-[10px]">var</Badge>
                      )}
                    </Button>
                  </TableCell>
                  <TableCell>
                    <Button variant="ghost" size="icon" onClick={() => removeRow(idx)} className="h-7 w-7">
                      <Trash2 className="h-3.5 w-3.5 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <Button variant="dashed" size="sm" onClick={addRow} className="w-full">
            <Plus className="h-3.5 w-3.5 mr-1" />
            Add variable
          </Button>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[200px]">Key</TableHead>
              <TableHead>Value</TableHead>
              <TableHead className="w-[80px]">Type</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {readOnlyRows.map((row) => (
              <TableRow key={row.key}>
                <TableCell className="font-mono text-xs font-medium">{row.key}</TableCell>
                <TableCell className="font-mono text-xs">
                  {row.isSecret ? (
                    <div className="flex items-center gap-2">
                      <span>{revealedSecrets.has(row.key) ? row.value : '••••••••'}</span>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => toggleReveal(row.key)}
                        className="h-6 w-6"
                      >
                        {revealedSecrets.has(row.key) ? (
                          <EyeOff className="h-3.5 w-3.5" />
                        ) : (
                          <Eye className="h-3.5 w-3.5" />
                        )}
                      </Button>
                    </div>
                  ) : (
                    <span>{String(row.value)}</span>
                  )}
                </TableCell>
                <TableCell>
                  {row.isSecret ? (
                    <Badge variant="destructive" className="text-[10px]">secret</Badge>
                  ) : (
                    <Badge variant="secondary" className="text-[10px]">var</Badge>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}
