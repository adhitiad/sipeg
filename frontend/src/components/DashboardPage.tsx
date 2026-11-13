import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { BarChart3, Building2, Users } from 'lucide-react'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import { useLocale, useMessages } from '@/i18n/locale-context'
import {
  fetchEmployeeSummary,
  resolveUnitNames,
  type EmployeeSummary,
} from '@/lib/dashboard'
import { useOrganizations } from '@/lib/organizations'
import { useSession } from '@/lib/session'

function useEmployeeSummary() {
  return useQuery({
    queryKey: ['employee-summary'] as const,
    queryFn: fetchEmployeeSummary,
    staleTime: 30_000,
    retry: false,
  })
}

function StatCard({
  label,
  value,
  hint,
}: {
  label: string
  value: string
  hint?: string
}) {
  return (
    <div className="island-shell rounded-2xl px-4 py-3">
      <dt className="island-kicker mb-1">{label}</dt>
      <dd className="m-0 text-xl font-semibold text-[var(--sea-ink)]">{value}</dd>
      {hint ? (
        <p className="m-0 mt-1 text-xs text-[var(--sea-ink-soft)]">{hint}</p>
      ) : null}
    </div>
  )
}

function StatusBar({ active, inactive }: { active: number; inactive: number }) {
  const total = active + inactive
  const activePct = total === 0 ? 0 : Math.round((active / total) * 100)
  const inactivePct = total === 0 ? 0 : 100 - activePct
  return (
    <div>
      <div
        className="flex h-3 w-full overflow-hidden rounded-full bg-[var(--surface-strong)]"
        role="img"
        aria-label={`Aktif ${activePct} persen, nonaktif ${inactivePct} persen`}
      >
        <span className="bg-[var(--lagoon)]" style={{ width: `${activePct}%` }} />
        <span className="bg-[var(--sea-ink-soft)]" style={{ width: `${inactivePct}%` }} />
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-4 text-xs text-[var(--sea-ink-soft)]">
        <span className="inline-flex items-center gap-1.5">
          <span className="h-2 w-2 rounded-full bg-[var(--lagoon)]" aria-hidden="true" />
          Aktif
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span
            className="h-2 w-2 rounded-full bg-[var(--sea-ink-soft)]"
            aria-hidden="true"
          />
          Nonaktif
        </span>
      </div>
    </div>
  )
}

function UnitBreakdown({
  entries,
}: {
  entries: Array<{ unitId: string | null; unitName: string; count: number }>
}) {
  const t = useMessages()
  const total = entries.reduce((sum, entry) => sum + entry.count, 0)
  const max = entries.reduce((maxValue, entry) => Math.max(maxValue, entry.count), 0)

  if (entries.length === 0) {
    return (
      <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('dashboard.emptyUnits')}
      </p>
    )
  }

  return (
    <div className="space-y-3">
      {entries.map((entry) => {
        const pct = total === 0 ? 0 : Math.round((entry.count / total) * 100)
        const width = max === 0 ? 0 : Math.round((entry.count / max) * 100)
        return (
          <div key={entry.unitId ?? 'none'}>
            <div className="flex items-baseline justify-between gap-3">
              <span className="text-sm text-[var(--sea-ink)]">{entry.unitName}</span>
              <span className="text-xs tabular-nums text-[var(--sea-ink-soft)]">
                {entry.count} ({pct}%)
              </span>
            </div>
            <div className="mt-1 h-2 w-full rounded-full bg-[var(--surface-strong)]">
              <div
                className="h-2 rounded-full bg-[var(--lagoon)]"
                style={{ width: `${width}%` }}
              />
            </div>
          </div>
        )
      })}
    </div>
  )
}

export default function DashboardPage() {
  const t = useMessages()
  const { locale } = useLocale()
  const session = useSession()
  const summaryQuery = useEmployeeSummary()
  const unitsQuery = useOrganizations()

  const permissions = session.data?.permissions ?? []
  const canReadEmployees =
    permissions.some((permission: string) => permission.startsWith('employee:read:')) ??
    false
  const canReadUnits =
    permissions.includes('organization:read') ||
    permissions.includes('organization:read:homebase')

  const summary: EmployeeSummary | undefined = summaryQuery.data
  const localeTag = locale === 'id' ? 'id-ID' : 'en-US'

  const unitEntries = useMemo(() => {
    if (!summary) {
      return []
    }
    return resolveUnitNames(summary, unitsQuery.data ?? [])
  }, [summary, unitsQuery.data])

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-3 flex items-center gap-2">
        <BarChart3 className="h-4 w-4" aria-hidden="true" />
        {t('dashboard.kicker')}
      </p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('dashboard.title')}
      </h1>
      <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('dashboard.lead')}
      </p>

      {summaryQuery.isPending ? (
        <div className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center">
          <LoadingBars size="md" decorative />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('dashboard.loadingTitle')}
          </p>
        </div>
      ) : summaryQuery.isError || !summary ? (
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('dashboard.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('dashboard.errorDescription')}
          </p>
          <RetryButton
            label={t('common.retry')}
            onClick={() => void summaryQuery.refetch()}
          />
        </div>
      ) : (
        <div className="grid gap-6">
          <section className="island-shell rounded-2xl p-6">
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold text-[var(--sea-ink)]">
              <Users className="h-5 w-5" aria-hidden="true" />
              {t('dashboard.employeeStatusTitle')}
            </h2>
            <dl className="grid gap-3 sm:grid-cols-3">
              <StatCard
                label={t('dashboard.totalEmployees')}
                value={new Intl.NumberFormat(localeTag).format(summary.total)}
              />
              <StatCard
                label={t('dashboard.activeEmployees')}
                value={new Intl.NumberFormat(localeTag).format(summary.active)}
              />
              <StatCard
                label={t('dashboard.inactiveEmployees')}
                value={new Intl.NumberFormat(localeTag).format(summary.inactive)}
              />
            </dl>
            <div className="mt-5">
              <StatusBar active={summary.active} inactive={summary.inactive} />
            </div>
          </section>

          {canReadUnits ? (
            <section className="island-shell rounded-2xl p-6">
              <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold text-[var(--sea-ink)]">
                <Building2 className="h-5 w-5" aria-hidden="true" />
                {t('dashboard.unitBreakdownTitle')}
              </h2>
              <UnitBreakdown entries={unitEntries} />
            </section>
          ) : null}
        </div>
      )}

      <p className="mt-6 text-xs text-[var(--sea-ink-soft)]">
        {t('dashboard.scopeNote')}
      </p>
      {!canReadEmployees ? (
        <p className="mt-2 text-xs text-[var(--sea-ink-soft)]">
          {t('dashboard.noAccessNote')}
        </p>
      ) : null}
    </div>
  )
}
