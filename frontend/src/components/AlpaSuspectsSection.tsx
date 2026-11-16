import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import { useMessages } from '@/i18n/locale-context'
import { fetchAlpaSuspects } from '@/lib/mesin-pin'
import type { AlpaSuspectsResponse } from '@/validations/schemas'

export default function AlpaSuspectsSection() {
  const t = useMessages()
  const now = new Date()
  const defaultUntil = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`

  const [report, setReport] = useState<AlpaSuspectsResponse | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: ({ until, months }: { until: string; months: number }) =>
      fetchAlpaSuspects(until, months),
    onSuccess: (data) => {
      setReport(data)
      setLoadError(null)
    },
    onError: (error: Error) => {
      setLoadError(error.message)
    },
  })

  const monthKeys = report?.items.length
    ? Object.keys(report.items[0]?.alpa ?? {}).sort()
    : []

  return (
    <div className="py-2">
      <div className="mb-3">
        <h2 className="m-0 text-lg font-semibold text-[var(--sea-ink)]">
          {t('mesinPin.suspectsTitle')}
        </h2>
        <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
          {t('mesinPin.suspectsLead')}
        </p>
      </div>

      <form
        className="mb-4 flex flex-wrap items-end gap-3"
        noValidate
        onSubmit={(event) => {
          event.preventDefault()
          const data = Object.fromEntries(
            new FormData(event.currentTarget).entries(),
          ) as Record<string, string>
          const months = Number(data.months)
          mutation.mutate({
            until: data.until ?? defaultUntil,
            months: Number.isFinite(months) ? months : 2,
          })
        }}
      >
        <div className="grid gap-1.5">
          <Label htmlFor="suspects-until">{t('mesinPin.until')}</Label>
          <Input
            id="suspects-until"
            name="until"
            type="month"
            required
            defaultValue={defaultUntil}
            className="h-11"
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="suspects-months">{t('mesinPin.months')}</Label>
          <Input
            id="suspects-months"
            name="months"
            type="number"
            min={2}
            max={12}
            step={1}
            required
            defaultValue={2}
            className="h-11 w-24"
          />
        </div>
        <Button type="submit" disabled={mutation.isPending}>
          {mutation.isPending ? t('mesinPin.loading') : t('mesinPin.loadReport')}
        </Button>
      </form>

      {loadError ? (
        <div className="py-6 text-center">
          <p className="mb-3 text-sm text-[var(--sea-ink-soft)]">{loadError}</p>
          <RetryButton
            label={t('common.retry')}
            onClick={() => {
              setLoadError(null)
            }}
          />
        </div>
      ) : null}

      {mutation.isPending && !report ? (
        <div className="py-8 text-center">
          <LoadingBars size="sm" label={t('mesinPin.suspectsTitle')} />
        </div>
      ) : null}

      {report && report.items.length === 0 ? (
        <p className="m-0 py-6 text-center text-sm text-[var(--sea-ink-soft)]">
          {t('mesinPin.suspectsEmpty')}
        </p>
      ) : null}

      {report && report.items.length > 0 ? (
        <div className="overflow-x-auto rounded-xl border border-[var(--line)]">
          <table className="min-w-full border-collapse text-sm">
            <caption className="sr-only">{t('mesinPin.suspectsTitle')}</caption>
            <thead>
              <tr className="text-left text-xs uppercase tracking-[0.16em] text-[var(--kicker)]">
                <th scope="col" className="px-3 py-2">
                  {t('mesinPin.colNama')}
                </th>
                {monthKeys.map((month) => (
                  <th key={month} scope="col" className="px-3 py-2 text-right">
                    {month}
                  </th>
                ))}
                <th scope="col" className="px-3 py-2 text-right">
                  {t('mesinPin.total')}
                </th>
              </tr>
            </thead>
            <tbody>
              {report.items.map((row) => (
                <tr
                  key={row.userId}
                  className="border-t border-[var(--line)] align-middle"
                >
                  <td className="px-3 py-2 font-medium">{row.nama}</td>
                  {monthKeys.map((month) => (
                    <td key={month} className="px-3 py-2 text-right tabular-nums">
                      {row.alpa[month] ?? 0}
                    </td>
                  ))}
                  <td className="px-3 py-2 text-right font-semibold tabular-nums">
                    {row.totalAlpa}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="m-0 border-t border-[var(--line)] bg-[var(--sand)] px-3 py-2 text-xs text-[var(--sea-ink-soft)]">
            <ShieldAlert
              className="mr-1 inline h-3.5 w-3.5 align-[-2px]"
              aria-hidden="true"
            />
            {t('mesinPin.suspectsNote')}
          </p>
        </div>
      ) : null}
    </div>
  )
}
