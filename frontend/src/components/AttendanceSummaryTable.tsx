import type { AttendanceSummaryDay } from '@/validations/schemas'
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

type AttendanceSummaryTableProps = {
  items: AttendanceSummaryDay[]
  isFetching: boolean
  onInspectDate?: (date: string) => void
  labels: {
    caption: string
    date: string
    total: string
    statusHadir: string
    statusIzin: string
    statusSakit: string
    statusAlpa: string
    statusCuti: string
    statusLibur: string
    grandTotal: string
    viewDetail?: string
  }
}

/**
 * Tabel ringkasan kehadiran unit per hari. Baris dibatasi enam kode status
 * baku (H, I, S, A, C, LD) dan satu kolom total; baris terakhir menjumlahkan
 * kolom pada rentang yang sedang ditampilkan.
 */
export default function AttendanceSummaryTable({
  items,
  isFetching,
  onInspectDate,
  labels,
}: AttendanceSummaryTableProps) {
  const totals = items.reduce(
    (acc, day) => ({
      H: acc.H + day.counts.H,
      I: acc.I + day.counts.I,
      S: acc.S + day.counts.S,
      A: acc.A + day.counts.A,
      C: acc.C + day.counts.C,
      LD: acc.LD + day.counts.LD,
    }),
    { H: 0, I: 0, S: 0, A: 0, C: 0, LD: 0 },
  )
  const grandTotal = Object.values(totals).reduce((sum, value) => sum + value, 0)

  const statusColumns = [
    {
      code: 'H',
      label: labels.statusHadir,
      value: (day: AttendanceSummaryDay) => day.counts.H,
    },
    {
      code: 'I',
      label: labels.statusIzin,
      value: (day: AttendanceSummaryDay) => day.counts.I,
    },
    {
      code: 'S',
      label: labels.statusSakit,
      value: (day: AttendanceSummaryDay) => day.counts.S,
    },
    {
      code: 'A',
      label: labels.statusAlpa,
      value: (day: AttendanceSummaryDay) => day.counts.A,
    },
    {
      code: 'C',
      label: labels.statusCuti,
      value: (day: AttendanceSummaryDay) => day.counts.C,
    },
    {
      code: 'LD',
      label: labels.statusLibur,
      value: (day: AttendanceSummaryDay) => day.counts.LD,
    },
  ] as const

  return (
    <div className="island-shell overflow-hidden rounded-2xl">
      <Table className="min-w-[46rem]">
        <TableCaption className="sr-only">{labels.caption}</TableCaption>
        <TableHeader>
          <TableRow className="border-[var(--line)] hover:bg-transparent">
            <TableHead scope="col" className="px-4 py-3 island-kicker">
              {labels.date}
            </TableHead>
            {statusColumns.map((column) => (
              <TableHead key={column.code} scope="col" className="px-4 py-3 text-right">
                <span className="island-kicker" title={column.label}>
                  {column.label}
                </span>
              </TableHead>
            ))}
            <TableHead scope="col" className="px-4 py-3 text-right island-kicker">
              {labels.total}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody aria-busy={isFetching}>
          {items.map((day) => {
            const dayTotal = Object.values(day.counts).reduce(
              (sum, value) => sum + value,
              0,
            )
            return (
              <TableRow key={day.date} className="border-[var(--line)] last:border-0">
                <TableCell className="px-4 py-3 tabular-nums text-[var(--sea-ink)]">
                  {onInspectDate ? (
                    <button
                      type="button"
                      onClick={() => onInspectDate(day.date)}
                      aria-label={
                        labels.viewDetail ? `${labels.viewDetail} ${day.date}` : day.date
                      }
                      className="inline-flex min-h-11 items-center rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 text-xs font-semibold text-[var(--sea-ink)] transition-colors hover:border-[var(--lagoon)] hover:text-[var(--sea-ink)] sm:min-h-9"
                    >
                      {day.date}
                    </button>
                  ) : (
                    day.date
                  )}
                </TableCell>
                {statusColumns.map((column) => (
                  <TableCell
                    key={column.code}
                    className="px-4 py-3 text-right tabular-nums text-[var(--sea-ink-soft)]"
                  >
                    {column.value(day)}
                  </TableCell>
                ))}
                <TableCell className="px-4 py-3 text-right tabular-nums font-semibold text-[var(--sea-ink)]">
                  {dayTotal}
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
        <TableFooter className="bg-transparent">
          <TableRow className="border-t border-[var(--line)]">
            <TableHead scope="row" className="px-4 py-3 text-left island-kicker">
              {labels.grandTotal}
            </TableHead>
            {statusColumns.map((column) => (
              <TableCell
                key={column.code}
                className="px-4 py-3 text-right tabular-nums font-semibold text-[var(--sea-ink)]"
              >
                {totals[column.code]}
              </TableCell>
            ))}
            <TableCell className="px-4 py-3 text-right tabular-nums font-semibold text-[var(--sea-ink)]">
              {grandTotal}
            </TableCell>
          </TableRow>
        </TableFooter>
      </Table>
    </div>
  )
}
