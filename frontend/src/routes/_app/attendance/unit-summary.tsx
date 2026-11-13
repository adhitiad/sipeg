import { useMemo, useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { ShieldAlert } from 'lucide-react'
import AttendanceSummaryTable from '@/components/AttendanceSummaryTable'
import LoadingRadar from '@/components/LoadingRadar'
import RetryButton from '@/components/RetryButton'
import { Input } from '@/components/ui/input'
import { useMessages } from '@/i18n/locale-context'
import { useAttendanceSummary } from '@/lib/attendance'
import { useSession } from '@/lib/session'

export const Route = createFileRoute('/_app/attendance/unit-summary')({
  component: UnitSummaryPage,
})

const MS_PER_DAY = 86_400_000

/** Tanggal hari ini dalam zona waktu operasional Asia/Jakarta, format YYYY-MM-DD. */
function jakartaToday(): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Jakarta',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date())
}

function daysBetween(from: string, to: string): number {
  const start = Date.parse(`${from}T00:00:00Z`)
  const end = Date.parse(`${to}T00:00:00Z`)
  return Math.round((end - start) / MS_PER_DAY) + 1
}

/**
 * Ringkasan kehadiran unit untuk atasan dan administrator. Cakupan unit
 * sepenuhnya ditegakkan backend (`attendance:read:any` lintas organisasi,
 * `attendance:read:homebase` terbatas pada unit pemohon beserta turunannya);
 * halaman ini tidak menawarkan filter unit karena data sudah dipersempit di
 * server.
 */
function UnitSummaryPage() {
  const t = useMessages()
  const session = useSession()
  const navigate = useNavigate()

  const permissions = session.data?.permissions ?? []
  const canReadAny = permissions.includes('attendance:read:any')
  const canReadHomebase = permissions.includes('attendance:read:homebase')
  const canRead = canReadAny || canReadHomebase

  const today = useMemo(() => jakartaToday(), [])
  const [from, setFrom] = useState(() => `${today.slice(0, 7)}-01`)
  const [to, setTo] = useState(today)

  const rangeDays = from && to ? daysBetween(from, to) : 0
  const rangeValid = from !== '' && to !== '' && from <= to && rangeDays <= 365

  const query = useAttendanceSummary({ from, to }, rangeValid)

  if (session.data && !canRead) {
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

  const scopeLabel = canReadAny ? t('employees.scopeAny') : t('employees.scopeHomebase')
  const data = query.data

  const labels = {
    caption: t('attendance.summaryCaption'),
    date: t('attendance.colDate'),
    total: t('attendance.summaryTotal'),
    statusHadir: t('attendance.statusHadir'),
    statusIzin: t('attendance.statusIzin'),
    statusSakit: t('attendance.statusSakit'),
    statusAlpa: t('attendance.statusAlpa'),
    statusCuti: t('attendance.statusCuti'),
    statusLibur: t('attendance.statusLibur'),
    grandTotal: t('attendance.summaryTotal'),
    viewDetail: t('attendance.viewDetail'),
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-3">{t('attendance.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('attendance.summaryTitle')}
      </h1>
      <p className="mb-2 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('attendance.summaryLead')}
      </p>
      <p className="mb-6 text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
        {scopeLabel}
      </p>

      <div className="mb-6 grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1">
          <label
            htmlFor="attendance-summary-from"
            className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
          >
            {t('attendance.fromLabel')}
          </label>
          <Input
            id="attendance-summary-from"
            type="date"
            value={from}
            max={today}
            onChange={(event) => setFrom(event.target.value)}
            className="h-11"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label
            htmlFor="attendance-summary-to"
            className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
          >
            {t('attendance.toLabel')}
          </label>
          <Input
            id="attendance-summary-to"
            type="date"
            value={to}
            max={today}
            onChange={(event) => setTo(event.target.value)}
            className="h-11"
          />
        </div>
      </div>

      {!rangeValid ? (
        <div role="alert" className="island-shell max-w-2xl rounded-2xl p-6">
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('attendance.rangeError')}
          </p>
        </div>
      ) : query.isPending ? (
        <div
          role="status"
          aria-live="polite"
          className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center"
        >
          <LoadingRadar size="md" label={t('attendance.summaryLoadingTitle')} />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('attendance.summaryLoadingTitle')}
          </p>
        </div>
      ) : query.isError || !data ? (
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('attendance.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('attendance.errorDescription')}
          </p>
          <RetryButton label={t('common.retry')} onClick={() => void query.refetch()} />
        </div>
      ) : data.length === 0 ? (
        <div className="island-shell max-w-2xl rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('attendance.summaryEmptyTitle')}
          </h2>
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('attendance.summaryEmptyDescription')}
          </p>
        </div>
      ) : (
        <AttendanceSummaryTable
          items={data}
          isFetching={query.isFetching}
          onInspectDate={(date) =>
            void navigate({ to: '/attendance/unit-records', search: { date } })
          }
          labels={labels}
        />
      )}
    </div>
  )
}
