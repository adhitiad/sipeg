import { useMemo, useState, type ChangeEvent } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { ShieldAlert } from 'lucide-react'
import AttendanceTable from '@/components/AttendanceTable'
import LoadingRadar from '@/components/LoadingRadar'
import RetryButton from '@/components/RetryButton'
import { Input } from '@/components/ui/input'
import { useLocale, useMessages } from '@/i18n/locale-context'
import { useAttendanceRecords, type AttendanceListParams } from '@/lib/attendance'
import { useSession } from '@/lib/session'

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
 * Riwayat kehadiran pegawai yang sedang masuk. Backend hanya menyajikan
 * catatan milik principal (`attendance:read:self`); tidak ada filter unit di
 * sini karena data yang tampil memang hanya milik pemanggil.
 */
export const Route = createFileRoute('/_app/attendance/my-records')({
  component: AttendanceRecordsPage,
})

function AttendanceRecordsPage() {
  const t = useMessages()
  const { locale } = useLocale()
  const session = useSession()

  const permissions = session.data?.permissions ?? []
  const canReadSelf = permissions.includes('attendance:read:self')

  const today = useMemo(() => jakartaToday(), [])
  const [from, setFrom] = useState(() => `${today.slice(0, 7)}-01`)
  const [to, setTo] = useState(today)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState<10 | 25 | 50 | 100>(20 as 10 | 25 | 50 | 100)
  const [search, setSearch] = useState('')
  const [sort, setSort] = useState<'date'>('date')
  const [order, setOrder] = useState<'asc' | 'desc'>('asc')
  const [status, setStatus] = useState<string | null>(null)

  const rangeDays = from && to ? daysBetween(from, to) : 0
  const rangeValid = from !== '' && to !== '' && from <= to && rangeDays <= 365

  const params: AttendanceListParams = useMemo(
    () => ({ page, pageSize, from, to, search, sort, order, status }),
    [page, pageSize, from, to, search, sort, order, status],
  )

  const query = useAttendanceRecords(params, rangeValid)

  function handleFromChange(value: string) {
    setFrom(value)
    setPage(1)
  }

  function handleToChange(value: string) {
    setTo(value)
    setPage(1)
  }

  function handleSortChange(sortField: 'date', nextOrder: 'asc' | 'desc') {
    setSort(sortField)
    setOrder(nextOrder)
    setPage(1)
  }

  if (session.data && !canReadSelf) {
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

  const data = query.data
  const localeTag = locale === 'id' ? 'id-ID' : 'en-US'
  const totalItems = data?.totalItems ?? 0
  const [rangeFrom, rangeTo] =
    totalItems === 0
      ? [0, 0]
      : [(page - 1) * pageSize + 1, Math.min(page * pageSize, totalItems)]

  const labels = {
    caption: t('attendance.tableCaption'),
    id: t('attendance.colId'),
    date: t('attendance.colDate'),
    status: t('attendance.colStatus'),
    note: t('attendance.colNote'),
    source: t('attendance.colSource'),
    resultLabel: t('attendance.resultCount', {
      from: new Intl.NumberFormat(localeTag).format(rangeFrom),
      to: new Intl.NumberFormat(localeTag).format(rangeTo),
      total: new Intl.NumberFormat(localeTag).format(totalItems),
    }),
    refreshingLabel: t('attendance.refreshing'),
    prevPage: t('attendance.prevPage'),
    nextPage: t('attendance.nextPage'),
    pageInfo: t('attendance.pageInfo', {
      page: String(page),
      total: String(Math.max(data?.totalPages ?? 1, 1)),
    }),
    searchLabel: t('attendance.teamSearchLabel'),
    searchPlaceholder: t('attendance.teamSearchPlaceholder'),
    statusLabel: t('attendance.colStatus'),
    statusAll: t('attendance.statusAll'),
    pageSizeLabel: t('attendance.pageSize'),
    itemsPerPage: t('attendance.teamResultCount', {
      from: new Intl.NumberFormat(localeTag).format(rangeFrom),
      to: new Intl.NumberFormat(localeTag).format(rangeTo),
      total: new Intl.NumberFormat(localeTag).format(totalItems),
    }),
    statusHadir: t('attendance.statusHadir'),
    statusIzin: t('attendance.statusIzin'),
    statusSakit: t('attendance.statusSakit'),
    statusAlpa: t('attendance.statusAlpa'),
    statusCuti: t('attendance.statusCuti'),
    statusLibur: t('attendance.statusLibur'),
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-3">{t('attendance.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('attendance.title')}
      </h1>
      <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('attendance.lead')}
      </p>

      <div className="mb-6 grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1">
          <label
            htmlFor="attendance-from"
            className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
          >
            {t('attendance.fromLabel')}
          </label>
          <Input
            id="attendance-from"
            type="date"
            value={from}
            max={today}
            onChange={(event: ChangeEvent<HTMLInputElement>) =>
              handleFromChange(event.target.value)
            }
            className="h-11"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label
            htmlFor="attendance-to"
            className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
          >
            {t('attendance.toLabel')}
          </label>
          <Input
            id="attendance-to"
            type="date"
            value={to}
            max={today}
            onChange={(event: ChangeEvent<HTMLInputElement>) =>
              handleToChange(event.target.value)
            }
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
          <LoadingRadar size="md" label={t('attendance.loadingTitle')} />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('attendance.loadingTitle')}
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
      ) : data.items.length === 0 ? (
        <div className="island-shell max-w-2xl rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('attendance.emptyTitle')}
          </h2>
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('attendance.emptyDescription')}
          </p>
        </div>
      ) : (
        <AttendanceTable
          items={data.items}
          page={page}
          pageSize={pageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isFetching={query.isFetching}
          search={search}
          sort={sort}
          order={order}
          status={status}
          onPageChange={setPage}
          onPageSizeChange={setPageSize}
          onSearchChange={setSearch}
          onSortChange={handleSortChange}
          onStatusChange={setStatus}
          labels={labels}
        />
      )}
    </div>
  )
}
