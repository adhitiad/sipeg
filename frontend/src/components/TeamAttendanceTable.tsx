import type { UnitDayRecord } from '@/validations/schemas'
import type { LocaleContextValue } from '@/lib/types/i18n'
import { useMessages } from '@/i18n/locale-context'
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

type Translator = LocaleContextValue['t']

type TeamAttendanceTableProps = {
  items: UnitDayRecord[]
  page: number
  pageSize: number
  totalPages: number
  totalItems: number
  recordedCount: number
  unrecordedCount: number
  isFetching: boolean
  onPageChange: (page: number) => void
  labels: {
    caption: string
    name: string
    nip: string
    unit: string
    status: string
    note: string
    noUnit: string
    noValue: string
    notRecorded: string
    teamCounts: string
    resultLabel: string
    refreshingLabel: string
    prevPage: string
    nextPage: string
    pageInfo: string
  }
}

function statusLabel(status: string, t: Translator): string {
  switch (status) {
    case 'H':
      return t('attendance.statusHadir')
    case 'I':
      return t('attendance.statusIzin')
    case 'S':
      return t('attendance.statusSakit')
    case 'A':
      return t('attendance.statusAlpa')
    case 'C':
      return t('attendance.statusCuti')
    case 'LD':
      return t('attendance.statusLibur')
    default:
      return status
  }
}

/**
 * Tabel kehadiran tim per tanggal. Satu baris per pegawai aktif dalam cakupan
 * atasan; `status` null berarti belum tercatat pada tanggal yang ditampilkan.
 * Pencarian dan penomoran halaman dikendalikan pemanggil; tabel hanya
 * menampilkan baris yang sudah disiapkan.
 */
export default function TeamAttendanceTable({
  items,
  page,
  pageSize,
  totalPages,
  totalItems,
  recordedCount,
  unrecordedCount,
  isFetching,
  onPageChange,
  labels,
}: TeamAttendanceTableProps) {
  const t = useMessages()

  const statusBadgeColor = (status: string): string => {
    switch (status) {
      case 'H':
        return 'text-[var(--sea-ink)]'
      case 'A':
        return 'text-destructive'
      default:
        return 'text-[var(--sea-ink-soft)]'
    }
  }

  const first = totalItems === 0 ? 0 : (page - 1) * pageSize + 1
  const last = totalItems === 0 ? 0 : Math.min(page * pageSize, totalItems)

  return (
    <div className="island-shell overflow-hidden rounded-2xl">
      <div className="border-b border-[var(--line)] px-4 py-3 text-sm">
        <p className="m-0 text-[var(--sea-ink-soft)]">
          {labels.teamCounts
            .replace('{recorded}', String(recordedCount))
            .replace('{unrecorded}', String(unrecordedCount))}
        </p>
      </div>
      <Table className="min-w-[44rem]">
        <TableCaption className="sr-only">{labels.caption}</TableCaption>
        <TableHeader>
          <TableRow className="border-[var(--line)] hover:bg-transparent">
            <TableHead scope="col" className="px-4 py-3 island-kicker">
              {labels.name}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3 island-kicker">
              {labels.nip}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3 island-kicker">
              {labels.unit}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3 island-kicker">
              {labels.status}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3 island-kicker">
              {labels.note}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody aria-busy={isFetching}>
          {items.map((record) => (
            <TableRow
              key={record.id}
              className="border-[var(--line)] last:border-0 hover:bg-transparent"
            >
              <TableHead
                scope="row"
                className="px-4 py-3 text-left font-semibold text-[var(--sea-ink)] whitespace-normal"
              >
                {record.nama}
              </TableHead>
              <TableCell className="px-4 py-3 font-mono text-xs text-[var(--sea-ink-soft)]">
                {record.nip ?? <span className="font-sans">{labels.noValue}</span>}
              </TableCell>
              <TableCell className="px-4 py-3 break-words text-[var(--sea-ink-soft)]">
                {record.unitName ?? <span>{labels.noUnit}</span>}
              </TableCell>
              <TableCell className="px-4 py-3">
                {record.status ? (
                  <span
                    className={`text-xs font-medium ${statusBadgeColor(record.status)}`}
                  >
                    {statusLabel(record.status, t)}
                  </span>
                ) : (
                  <span className="text-xs text-[var(--sea-ink-soft)]">
                    {labels.notRecorded}
                  </span>
                )}
              </TableCell>
              <TableCell className="px-4 py-3 text-sm break-words text-[var(--sea-ink-soft)]">
                {record.note ?? '-'}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <div className="flex flex-col gap-3 border-t border-[var(--line)] px-4 py-3 text-sm sm:flex-row sm:items-center sm:justify-between">
        <p className="m-0 text-[var(--sea-ink-soft)]">
          {labels.resultLabel
            .replace('{from}', String(first))
            .replace('{to}', String(last))
            .replace('{total}', String(totalItems))}
          {isFetching ? ` · ${labels.refreshingLabel}` : ''}
        </p>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
            className="h-11 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-4 text-xs font-semibold text-[var(--sea-ink)] disabled:cursor-not-allowed disabled:opacity-50 sm:h-9"
          >
            {labels.prevPage}
          </button>
          <span className="text-xs text-[var(--sea-ink-soft)]">
            {labels.pageInfo
              .replace('{page}', String(page))
              .replace('{total}', String(Math.max(totalPages, 1)))}
          </span>
          <button
            type="button"
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages}
            className="h-11 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-4 text-xs font-semibold text-[var(--sea-ink)] disabled:cursor-not-allowed disabled:opacity-50 sm:h-9"
          >
            {labels.nextPage}
          </button>
        </div>
      </div>
    </div>
  )
}
