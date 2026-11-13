import { useState } from 'react'
import { Activity, ShieldAlert } from 'lucide-react'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useLocale, useMessages } from '@/i18n/locale-context'
import { SLO_LOOKBACKS, useSloMetrics, type SloLookbackHours } from '@/lib/slo'
import { useSession } from '@/lib/session'

function formatCount(value: number, locale: string): string {
  return new Intl.NumberFormat(locale).format(value)
}

function formatPercent(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, {
    style: 'percent',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
}

function formatMs(value: number): string {
  return `${Number.isInteger(value) ? value : value.toFixed(1)} ms`
}

function formatWindowTime(value: string, locale: string): string {
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}

/**
 * Halaman admin metrik SLO: ringkasan permintaan, latensi persentil, dan
 * laju galat per rute pada rentang yang dipilih. Data bersifat agregat;
 * metrik mentah per jendela tidak pernah meninggalkan server.
 */
export default function SloMetricsPage() {
  const t = useMessages()
  const { locale } = useLocale()
  const session = useSession()
  const [hours, setHours] = useState<SloLookbackHours>(24)
  const query = useSloMetrics(hours)

  const permissions = session.data?.permissions ?? []
  if (session.data && !permissions.includes('slo:read')) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <ShieldAlert
            className="mx-auto mb-3 h-8 w-8 text-[var(--sea-ink-soft)]"
            aria-hidden="true"
          />
          <h1 className="display-title mb-2 text-2xl font-bold text-[var(--sea-ink)]">
            {t('state.noAccess.title')}
          </h1>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('state.noAccess.description')}
          </p>
        </section>
      </div>
    )
  }

  const localeTag = locale === 'id' ? 'id-ID' : 'en-US'
  const data = query.data

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-3 flex items-center gap-2">
        <Activity className="h-4 w-4" aria-hidden="true" />
        {t('slo.kicker')}
      </p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('slo.title')}
      </h1>
      <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('slo.lead')}
      </p>

      <div
        role="group"
        aria-label={t('slo.rangeLabel')}
        className="mb-6 inline-flex items-center rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] p-1"
      >
        {SLO_LOOKBACKS.map((option) => (
          <button
            key={option}
            type="button"
            aria-pressed={hours === option}
            onClick={() => setHours(option)}
            className={
              hours === option
                ? 'h-9 rounded-full px-4 text-xs font-bold text-[var(--sea-ink)] shadow-[0_0_0_1px_var(--chip-line)_inset]'
                : 'h-9 rounded-full px-4 text-xs font-semibold text-[var(--sea-ink-soft)] transition hover:text-[var(--sea-ink)]'
            }
          >
            {option === 24 ? t('slo.range24h') : t('slo.range7d')}
          </button>
        ))}
      </div>

      {query.isPending ? (
        <div
          role="status"
          aria-live="polite"
          className="island-shell flex items-center gap-3 rounded-2xl p-6 text-sm text-[var(--sea-ink-soft)]"
        >
          <LoadingBars size="sm" decorative />
          {t('common.loading')}
        </div>
      ) : query.isError || !data ? (
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('slo.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('slo.errorDescription')}
          </p>
          <RetryButton label={t('common.retry')} onClick={() => void query.refetch()} />
        </div>
      ) : data.requests === 0 ? (
        <section className="island-shell max-w-2xl rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('slo.emptyTitle')}
          </h2>
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('slo.emptyDescription')}
          </p>
        </section>
      ) : (
        <>
          <p className="mb-4 text-xs text-[var(--sea-ink-soft)]">
            {t('slo.window', {
              start: formatWindowTime(data.windowStart, localeTag),
              end: formatWindowTime(data.windowEnd, localeTag),
            })}
          </p>
          <dl className="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-5">
            {[
              {
                label: t('slo.totalRequests'),
                value: formatCount(data.requests, localeTag),
              },
              { label: t('slo.totalErrors'), value: formatCount(data.errors, localeTag) },
              {
                label: t('slo.errorRate'),
                value: formatPercent(data.errorRate, localeTag),
              },
              { label: 'P95', value: formatMs(data.p95Ms) },
              { label: 'P99', value: formatMs(data.p99Ms) },
            ].map((card) => (
              <div key={card.label} className="island-shell rounded-2xl px-4 py-3">
                <dt className="island-kicker mb-1">{card.label}</dt>
                <dd className="m-0 text-xl font-semibold text-[var(--sea-ink)]">
                  {card.value}
                </dd>
              </div>
            ))}
          </dl>
          <div className="island-shell rounded-2xl">
            <Table className="min-w-[56rem]">
              <TableCaption className="sr-only">{t('slo.tableCaption')}</TableCaption>
              <TableHeader>
                <TableRow className="border-[var(--line)] hover:bg-transparent">
                  <TableHead scope="col" className="px-4 py-3">
                    {t('slo.colRoute')}
                  </TableHead>
                  <TableHead scope="col" className="px-4 py-3">
                    {t('slo.colMethod')}
                  </TableHead>
                  <TableHead scope="col" className="px-4 py-3 text-right">
                    {t('slo.colRequests')}
                  </TableHead>
                  <TableHead scope="col" className="px-4 py-3 text-right">
                    {t('slo.colErrors')}
                  </TableHead>
                  <TableHead scope="col" className="px-4 py-3 text-right">
                    {t('slo.colErrorRate')}
                  </TableHead>
                  <TableHead scope="col" className="px-4 py-3 text-right">
                    {t('slo.colAvg')}
                  </TableHead>
                  <TableHead scope="col" className="px-4 py-3 text-right">
                    P95
                  </TableHead>
                  <TableHead scope="col" className="px-4 py-3 text-right">
                    P99
                  </TableHead>
                  <TableHead scope="col" className="px-4 py-3 text-right">
                    {t('slo.colMax')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.routes.map((route) => (
                  <TableRow
                    key={`${route.method} ${route.route}`}
                    className="border-[var(--line)] last:border-0 hover:bg-transparent"
                  >
                    <TableHead
                      scope="row"
                      className="px-4 py-3 text-left font-mono text-xs font-medium break-all"
                    >
                      {route.route}
                    </TableHead>
                    <TableCell className="px-4 py-3 font-mono text-xs">
                      {route.method}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right tabular-nums">
                      {formatCount(route.requests, localeTag)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right tabular-nums">
                      {formatCount(route.errors, localeTag)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right tabular-nums">
                      {formatPercent(route.errorRate, localeTag)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right tabular-nums">
                      {formatMs(route.avgMs)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right tabular-nums">
                      {formatMs(route.p95Ms)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right tabular-nums">
                      {formatMs(route.p99Ms)}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right tabular-nums">
                      {formatMs(route.maxMs)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </>
      )}
    </div>
  )
}
