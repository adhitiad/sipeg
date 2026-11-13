import { useMemo, useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { ShieldAlert } from 'lucide-react'
import LoadingRadar from '@/components/LoadingRadar'
import TeamAttendanceTable from '@/components/TeamAttendanceTable'
import RetryButton from '@/components/RetryButton'
import { Input } from '@/components/ui/input'
import { useMessages } from '@/i18n/locale-context'
import { TEAM_ATTENDANCE_PAGE_SIZE, useUnitDay } from '@/lib/attendance'
import { useSession } from '@/lib/session'

const DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/

/** Tanggal hari ini dalam zona waktu operasional Asia/Jakarta, format YYYY-MM-DD. */
function jakartaToday(): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Jakarta',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date())
}

function isValidDateString(value: string): boolean {
  if (!DATE_PATTERN.test(value)) {
    return false
  }
  return !Number.isNaN(Date.parse(`${value}T00:00:00Z`))
}

type UnitRecordsSearch = { date?: string }

export const Route = createFileRoute('/_app/attendance/unit-records')({
  validateSearch: (search: Record<string, unknown>): UnitRecordsSearch => {
    const raw = search.date
    if (typeof raw === 'string' && isValidDateString(raw)) {
      return { date: raw }
    }
    return {}
  },
  component: UnitRecordsPage,
})

/**
 * Detail kehadiran tim per pegawai untuk satu tanggal. Cakupan sepenuhnya
 * ditegakkan backend (`attendance:read:any` lintas organisasi,
 * `attendance:read:homebase` terbatas pada unit pemohon beserta turunannya);
 * pegawai tanpa catatan tampil sebagai belum tercatat, bukan alpa.
 */
function UnitRecordsPage() {
  const t = useMessages()
  const session = useSession()
  const navigate = useNavigate({ from: Route.fullPath })
  const searchParams = Route.useSearch()
  const today = useMemo(() => jakartaToday(), [])
  const date = searchParams.date ?? today

  const permissions = session.data?.permissions ?? []
  const canReadAny = permissions.includes('attendance:read:any')
  const canReadHomebase = permissions.includes('attendance:read:homebase')
  const canRead = canReadAny || canReadHomebase

  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)

  const query = useUnitDay(date, canRead)

  const filtered = useMemo(() => {
    const rows = query.data ?? []
    const term = search.trim().toLowerCase()
    if (term === '') {
      return rows
    }
    return rows.filter(
      (row) =>
        row.nama.toLowerCase().includes(term) ||
        (row.nip ?? '').toLowerCase().includes(term),
    )
  }, [query.data, search])

  const totalItems = filtered.length
  const totalPages = Math.max(Math.ceil(totalItems / TEAM_ATTENDANCE_PAGE_SIZE), 1)
  const currentPage = Math.min(page, totalPages)
  const pageItems = filtered.slice(
    (currentPage - 1) * TEAM_ATTENDANCE_PAGE_SIZE,
    currentPage * TEAM_ATTENDANCE_PAGE_SIZE,
  )
  const recordedCount = filtered.filter((row) => row.status !== null).length
  const unrecordedCount = totalItems - recordedCount

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
    caption: t('attendance.teamCaption'),
    name: t('employees.colName'),
    nip: t('employees.colNip'),
    unit: t('employees.colUnit'),
    status: t('attendance.colStatus'),
    note: t('attendance.colNote'),
    noUnit: t('employees.noUnit'),
    noValue: t('employees.noValue'),
    notRecorded: t('attendance.teamNotRecorded'),
    teamCounts: t('attendance.teamCounts'),
    resultLabel: t('attendance.teamResultCount'),
    refreshingLabel: t('attendance.refreshing'),
    prevPage: t('attendance.prevPage'),
    nextPage: t('attendance.nextPage'),
    pageInfo: t('attendance.pageInfo'),
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-3">{t('attendance.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('attendance.teamTitle')}
      </h1>
      <p className="mb-2 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('attendance.teamLead')}
      </p>
      <p className="mb-6 text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
        {scopeLabel}
      </p>

      <div className="mb-6 grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1">
          <label
            htmlFor="attendance-team-date"
            className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
          >
            {t('attendance.teamDateLabel')}
          </label>
          <Input
            id="attendance-team-date"
            type="date"
            value={date}
            max={today}
            onChange={(event) => {
              const next = event.target.value
              if (isValidDateString(next)) {
                setPage(1)
                void navigate({ search: { date: next }, replace: true })
              }
            }}
            className="h-11"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label
            htmlFor="attendance-team-search"
            className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
          >
            {t('attendance.teamSearchLabel')}
          </label>
          <Input
            id="attendance-team-search"
            type="search"
            value={search}
            onChange={(event) => {
              setSearch(event.target.value)
              setPage(1)
            }}
            placeholder={t('attendance.teamSearchPlaceholder')}
            className="h-11"
          />
        </div>
      </div>

      {query.isPending ? (
        <div
          role="status"
          aria-live="polite"
          className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center"
        >
          <LoadingRadar size="md" label={t('attendance.teamLoadingTitle')} />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('attendance.teamLoadingTitle')}
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
            {t('attendance.teamEmptyTitle')}
          </h2>
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('attendance.teamEmptyDescription')}
          </p>
        </div>
      ) : totalItems === 0 ? (
        <div className="island-shell max-w-2xl rounded-2xl p-6">
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('attendance.teamNoMatch')}
          </p>
        </div>
      ) : (
        <TeamAttendanceTable
          items={pageItems}
          page={currentPage}
          pageSize={TEAM_ATTENDANCE_PAGE_SIZE}
          totalPages={totalPages}
          totalItems={totalItems}
          recordedCount={recordedCount}
          unrecordedCount={unrecordedCount}
          isFetching={query.isFetching}
          onPageChange={setPage}
          labels={labels}
        />
      )}
    </div>
  )
}
