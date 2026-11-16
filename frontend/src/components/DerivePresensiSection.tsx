import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { CalendarCheck, Info } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useMessages } from '@/i18n/locale-context'
import { derivePresensi, upsertPresensi } from '@/lib/mesin-pin'
import type { DerivePresensiResponse } from '@/validations/schemas'

const STATUS_OPTIONS = ['H', 'I', 'S', 'A', 'C', 'LD'] as const

export default function DerivePresensiSection() {
  const t = useMessages()
  const today = new Date()
  const defaultFrom = new Date(today.getFullYear(), today.getMonth(), 1)
    .toISOString()
    .slice(0, 10)
  const defaultTo = today.toISOString().slice(0, 10)

  const [summary, setSummary] = useState<DerivePresensiResponse | null>(null)
  const [deriveError, setDeriveError] = useState<string | null>(null)

  const deriveMutation = useMutation({
    mutationFn: ({ from, to }: { from: string; to: string }) => derivePresensi(from, to),
    onSuccess: (result) => {
      setSummary(result)
      setDeriveError(null)
      if (result.auditFailed) {
        toast.warning(t('mesinPin.auditFailed'))
      } else {
        toast.success(
          t('mesinPin.deriveDone', {
            created: String(result.created),
            updated: String(result.updated),
          }),
        )
      }
    },
    onError: (error: Error) => {
      setDeriveError(error.message)
      toast.error(t('mesinPin.deriveError'))
    },
  })

  const manualMutation = useMutation({
    mutationFn: upsertPresensi,
    onSuccess: (result) => {
      toast.success(
        result.created ? t('mesinPin.manualCreated') : t('mesinPin.manualUpdated'),
      )
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  return (
    <div className="grid gap-6 py-2 lg:grid-cols-2">
      <section aria-labelledby="derive-title" className="grid content-start gap-3">
        <div>
          <h2
            id="derive-title"
            className="m-0 text-lg font-semibold text-[var(--sea-ink)]"
          >
            {t('mesinPin.deriveTitle')}
          </h2>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('mesinPin.deriveLead')}
          </p>
        </div>
        <div
          role="note"
          className="flex gap-2 rounded-xl border border-[var(--line)] bg-[var(--sand)] p-3 text-xs text-[var(--sea-ink-soft)]"
        >
          <Info className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
          <span>{t('mesinPin.deriveNote')}</span>
        </div>
        <form
          className="grid gap-3"
          noValidate
          onSubmit={(event) => {
            event.preventDefault()
            const data = Object.fromEntries(
              new FormData(event.currentTarget).entries(),
            ) as Record<string, string>
            deriveMutation.mutate({ from: data.from ?? '', to: data.to ?? '' })
          }}
        >
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-1.5">
              <Label htmlFor="derive-from">{t('mesinPin.from')}</Label>
              <Input
                id="derive-from"
                name="from"
                type="date"
                required
                defaultValue={defaultFrom}
                className="h-11"
              />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="derive-to">{t('mesinPin.to')}</Label>
              <Input
                id="derive-to"
                name="to"
                type="date"
                required
                defaultValue={defaultTo}
                className="h-11"
              />
            </div>
          </div>
          {deriveError ? (
            <p role="alert" className="m-0 text-sm text-destructive">
              {deriveError}
            </p>
          ) : null}
          <div>
            <Button type="submit" disabled={deriveMutation.isPending}>
              <CalendarCheck className="mr-1.5 h-4 w-4" aria-hidden="true" />
              {deriveMutation.isPending
                ? t('mesinPin.deriving')
                : t('mesinPin.deriveButton')}
            </Button>
          </div>
        </form>
        {summary ? (
          <div
            role="status"
            className="rounded-xl border border-[var(--line)] bg-[var(--card)] p-4 text-sm"
          >
            <p className="m-0 font-semibold text-[var(--sea-ink)]">
              {t('mesinPin.deriveSummary', {
                from: summary.from,
                to: summary.to,
              })}
            </p>
            <dl className="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-4">
              <div>
                <dt className="text-xs text-[var(--sea-ink-soft)]">
                  {t('mesinPin.labelCreated')}
                </dt>
                <dd className="m-0 font-semibold tabular-nums">{summary.created}</dd>
              </div>
              <div>
                <dt className="text-xs text-[var(--sea-ink-soft)]">
                  {t('mesinPin.labelUpdated')}
                </dt>
                <dd className="m-0 font-semibold tabular-nums">{summary.updated}</dd>
              </div>
              <div>
                <dt className="text-xs text-[var(--sea-ink-soft)]">
                  {t('mesinPin.skippedManual')}
                </dt>
                <dd className="m-0 font-semibold tabular-nums">
                  {summary.skippedManual}
                </dd>
              </div>
              <div>
                <dt className="text-xs text-[var(--sea-ink-soft)]">
                  {t('mesinPin.audit')}
                </dt>
                <dd className="m-0 font-semibold">
                  {summary.auditFailed
                    ? t('mesinPin.auditFailedShort')
                    : t('mesinPin.auditOk')}
                </dd>
              </div>
            </dl>
            {Object.keys(summary.byStatus).length > 0 ? (
              <p className="m-0 mt-2 text-xs text-[var(--sea-ink-soft)]">
                {Object.entries(summary.byStatus)
                  .map(([status, count]) => `${status}: ${count}`)
                  .join(' · ')}
              </p>
            ) : null}
          </div>
        ) : null}
      </section>

      <section aria-labelledby="manual-title" className="grid content-start gap-3">
        <div>
          <h2
            id="manual-title"
            className="m-0 text-lg font-semibold text-[var(--sea-ink)]"
          >
            {t('mesinPin.manualTitle')}
          </h2>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('mesinPin.manualLead')}
          </p>
        </div>
        <form
          className="grid gap-3"
          noValidate
          onSubmit={(event) => {
            event.preventDefault()
            const data = Object.fromEntries(
              new FormData(event.currentTarget).entries(),
            ) as Record<string, string>
            manualMutation.mutate({
              userId: data.userId?.trim() ?? '',
              tanggal: data.tanggal ?? '',
              status: (data.status ?? 'H') as 'H' | 'I' | 'S' | 'A' | 'C' | 'LD',
              keterangan: data.keterangan?.trim() ? data.keterangan.trim() : null,
            })
          }}
        >
          <div className="grid gap-1.5">
            <Label htmlFor="upsert-userId">{t('mesinPin.userId')}</Label>
            <Input
              id="upsert-userId"
              name="userId"
              required
              className="h-11 font-mono text-xs"
              placeholder="00000000-0000-4000-8000-..."
              autoComplete="off"
            />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-1.5">
              <Label htmlFor="upsert-tanggal">{t('mesinPin.tanggal')}</Label>
              <Input
                id="upsert-tanggal"
                name="tanggal"
                type="date"
                required
                className="h-11"
              />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="upsert-status">{t('mesinPin.status')}</Label>
              <Select name="status" defaultValue="H">
                <SelectTrigger id="upsert-status" className="h-11">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {STATUS_OPTIONS.map((status) => (
                    <SelectItem key={status} value={status}>
                      {status}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="upsert-keterangan">{t('mesinPin.keterangan')}</Label>
            <Input
              id="upsert-keterangan"
              name="keterangan"
              maxLength={500}
              className="h-11"
              autoComplete="off"
            />
          </div>
          <div>
            <Button type="submit" disabled={manualMutation.isPending}>
              {manualMutation.isPending
                ? t('payrollScale.saving')
                : t('mesinPin.manualSave')}
            </Button>
          </div>
        </form>
      </section>
    </div>
  )
}
