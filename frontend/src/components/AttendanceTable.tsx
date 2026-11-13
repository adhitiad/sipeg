import type { AttendanceRecord } from '@/validations/schemas'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import SortIcon from '@/components/SortIcon'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'

const PAGE_SIZES = [10, 25, 50, 100] as const
export type AttendancePageSize = (typeof PAGE_SIZES)[number]

type AttendanceTableProps = {
  items: AttendanceRecord[]
  page: number
  pageSize: AttendancePageSize
  totalPages: number
  totalItems: number
  isFetching: boolean
  search: string
  sort: 'date'
  order: 'asc' | 'desc'
  status: string | null
  onPageChange: (page: number) => void
  onPageSizeChange: (size: AttendancePageSize) => void
  onSearchChange: (value: string) => void
  onSortChange: (sort: 'date', order: 'asc' | 'desc') => void
  onStatusChange: (value: string | null) => void
  labels: {
    caption: string
    id: string
    date: string
    status: string
    note: string
    source: string
    resultLabel: string
    refreshingLabel: string
    prevPage: string
    nextPage: string
    pageInfo: string
    searchLabel: string
    searchPlaceholder: string
    statusLabel: string
    statusAll: string
    pageSizeLabel: string
    itemsPerPage: string
    statusHadir: string
    statusIzin: string
    statusSakit: string
    statusAlpa: string
    statusCuti: string
    statusLibur: string
  }
}

function statusLabel(status: string, t: (key: string) => string): string {
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

function statusBadgeColor(status: string): string {
  switch (status) {
    case 'H':
      return 'text-[var(--sea-ink)]'
    case 'I':
    case 'S':
    case 'C':
    case 'LD':
      return 'text-[var(--sea-ink-soft)]'
    case 'A':
      return 'text-destructive'
    default:
      return 'text-[var(--sea-ink-soft)]'
  }
}

function pageRange(totalItems: number, page: number, pageSize: number): [number, number] {
  if (totalItems === 0) {
    return [0, 0]
  }
  const from = (page - 1) * pageSize + 1
  const to = Math.min(page * pageSize, totalItems)
  return [from, to]
}

export default function AttendanceTable({
  items,
  page,
  pageSize,
  totalPages,
  totalItems,
  isFetching,
  search,
  sort,
  order,
  status,
  onPageChange,
  onPageSizeChange,
  onSearchChange,
  onSortChange,
  onStatusChange,
  labels,
}: AttendanceTableProps) {
  const t = useMessages()
  const [from, to] = pageRange(totalItems, page, pageSize)

  return (
    <div className="island-shell overflow-hidden rounded-2xl">
      <div className="border-b border-[var(--line)] px-4 py-3 sm:flex sm:items-center sm:justify-between gap-3">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center">
          <label
            htmlFor="attendance-search"
            className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
          >
            {labels.searchLabel}
          </label>
          <Input
            id="attendance-search"
            type="search"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder={labels.searchPlaceholder}
            className="h-11 w-full sm:w-64"
          />
        </div>

        <div className="flex flex-col gap-1 sm:flex-row sm:items-center">
          <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
            {labels.statusLabel}
          </span>
          <Select
            value={status ?? 'all'}
            onValueChange={(value) => onStatusChange(value === 'all' ? null : value)}
          >
            <SelectTrigger className="h-11 min-w-[14rem]" aria-label={labels.statusLabel}>
              <SelectValue placeholder={labels.statusAll} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{labels.statusAll}</SelectItem>
              <SelectItem value="H">{labels.statusHadir}</SelectItem>
              <SelectItem value="I">{labels.statusIzin}</SelectItem>
              <SelectItem value="S">{labels.statusSakit}</SelectItem>
              <SelectItem value="A">{labels.statusAlpa}</SelectItem>
              <SelectItem value="C">{labels.statusCuti}</SelectItem>
              <SelectItem value="LD">{labels.statusLibur}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1 sm:flex-row sm:items-center">
          <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
            {labels.pageSizeLabel}
          </span>
          <Select
            value={String(pageSize)}
            onValueChange={(value) =>
              onPageSizeChange(value as unknown as AttendancePageSize)
            }
          >
            <SelectTrigger className="h-11 w-20" aria-label={labels.pageSizeLabel}>
              <SelectValue placeholder={String(pageSize)} />
            </SelectTrigger>
            <SelectContent>
              {PAGE_SIZES.map((size) => (
                <SelectItem key={size} value={String(size)}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <Table className="min-w-[40rem]">
        <TableCaption className="sr-only">{labels.caption}</TableCaption>
        <TableHeader>
          <TableRow className="border-[var(--line)] hover:bg-transparent">
            <TableHead scope="col" className="px-4 py-3">
              {labels.id}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              <button
                type="button"
                onClick={() =>
                  onSortChange(
                    'date',
                    sort === 'date' ? (order === 'asc' ? 'desc' : 'asc') : 'asc',
                  )
                }
                aria-label={labels.date}
                className="island-kicker inline-flex items-center gap-1"
              >
                {labels.date}
                <SortIcon active={sort === 'date'} order={order} />
              </button>
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.status}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.note}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.source}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody aria-busy={isFetching}>
          {items.map((record) => (
            <TableRow key={record.id} className="border-[var(--line)] last:border-0">
              <TableCell className="px-4 py-3 font-mono text-xs break-all">
                {record.id}
              </TableCell>
              <TableCell className="px-4 py-3 text-sm text-[var(--sea-ink)] break-all">
                {record.date}
              </TableCell>
              <TableCell className="px-4 py-3">
                <span
                  className={`font-medium text-xs capitalize ${statusBadgeColor(record.status)}`}
                >
                  {statusLabel(record.status, (key: string) => t(key as MessageKey))}
                </span>
              </TableCell>
              <TableCell className="px-4 py-3 text-sm text-[var(--sea-ink-soft)] break-words">
                {record.note ?? '-'}
              </TableCell>
              <TableCell className="px-4 py-3 text-sm font-mono text-[var(--sea-ink-soft)] break-all">
                {record.source ?? '-'}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <div className="flex flex-col gap-3 border-t border-[var(--line)] px-4 py-3 text-sm sm:flex-row sm:items-center sm:justify-between">
        <p className="m-0 text-[var(--sea-ink-soft)]">
          {labels.itemsPerPage
            .replace('{from}', String(from))
            .replace('{to}', String(to))
            .replace('{total}', String(totalItems))}
          {isFetching ? ` · ${labels.refreshingLabel}` : ''}
        </p>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
            aria-label={labels.prevPage}
            className="h-11 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-4 text-xs font-semibold text-[var(--sea-ink)] disabled:cursor-not-allowed disabled:opacity-50 sm:h-9"
          >
            <ChevronLeft className="h-4 w-4" aria-hidden="true" />
          </button>
          {[...Array(Math.max(totalPages, 1))].map((p) => {
            const n = p + 1
            const active = n === page
            return (
              <button
                key={n}
                type="button"
                onClick={() => onPageChange(n)}
                aria-label={n === page ? 'Halaman ' + String(n) : undefined}
                aria-current={active ? 'page' : undefined}
                className={
                  active
                    ? 'h-10 w-10 rounded-full bg-[var(--lagoon)] text-white shadow-[0_0_0_1px_var(--lagoon)_inset] text-xs font-semibold'
                    : 'h-10 w-10 rounded-full border border-[var(--line)] bg-[var(--chip-bg)] text-[var(--sea-ink)] text-xs font-semibold hover:bg-[var(--surface-strong)] disabled:opacity-50'
                }
              >
                {String(n)}
              </button>
            )
          })}
          <button
            type="button"
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages}
            aria-label={labels.nextPage}
            className="h-11 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-4 text-xs font-semibold text-[var(--sea-ink)] disabled:cursor-not-allowed disabled:opacity-50 sm:h-9"
          >
            <ChevronRight className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <span className="text-xs text-[var(--sea-ink-soft)]">
          {labels.pageInfo
            .replace('{page}', String(page))
            .replace('{total}', String(Math.max(totalPages, 1)))}
        </span>
      </div>
    </div>
  )
}
