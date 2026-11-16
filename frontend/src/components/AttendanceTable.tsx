import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Search } from 'lucide-react'
import { DataTable } from '@/components/DataTable'
import type { DataTableLabels } from '@/components/DataTable'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useMessages } from '@/i18n/locale-context'
import {
  ATTENDANCE_SORT_FIELDS,
  type AttendanceSortField,
  type AttendanceSortOrder,
} from '@/lib/attendance'
import type { MessageKey } from '@/i18n/dictionary'
import type { AttendanceRecord } from '@/validations/schemas'

const PAGE_SIZES = [10, 25, 50, 100] as const
export type AttendancePageSize = (typeof PAGE_SIZES)[number]

type AttendanceTableProps = {
  items: AttendanceRecord[]
  page: number
  pageSize: AttendancePageSize
  totalPages: number
  totalItems: number
  isFetching: boolean
  // Pencarian hanya dipasang pada tampilan unit. Pada riwayat pribadi kontrol
  // ini dihilangkan karena cakupan endpoint sudah milik principal sendiri.
  search?: string
  sort: AttendanceSortField
  order: AttendanceSortOrder
  status: string | null
  onPageChange: (page: number) => void
  onPageSizeChange: (size: AttendancePageSize) => void
  onSearchChange?: (value: string) => void
  onSortChange: (field: AttendanceSortField, order: AttendanceSortOrder) => void
  onStatusChange: (value: string | null) => void
  labels: DataTableLabels & {
    id: string
    date: string
    status: string
    note: string
    source: string
    searchLabel?: string
    searchPlaceholder?: string
    statusLabel: string
    statusAll: string
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

/**
 * Tabel riwayat presensi pribadi. Backend memegang pencarian, penyaringan
 * status, pengurutan, dan pemotongan baris; tabel ini meneruskan pilihan
 * pengguna ke mesin tabel bersama.
 */
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
  const showSearch = typeof onSearchChange === 'function'

  const columns = useMemo<ColumnDef<AttendanceRecord, unknown>[]>(
    () => [
      {
        id: 'id',
        accessorFn: (row) => row.id,
        header: labels.id,
        cell: ({ row }) => (
          <span className="font-mono text-xs break-all">{row.original.id}</span>
        ),
      },
      {
        id: 'tanggal',
        accessorFn: (row) => row.date,
        header: labels.date,
        cell: ({ row }) => (
          <span className="text-sm break-all text-[var(--sea-ink)]">
            {row.original.date}
          </span>
        ),
      },
      {
        id: 'status',
        accessorFn: (row) => row.status,
        header: labels.status,
        cell: ({ row }) => (
          <span
            className={`text-xs font-medium capitalize ${statusBadgeColor(
              row.original.status,
            )}`}
          >
            {statusLabel(row.original.status, (key) => t(key as MessageKey))}
          </span>
        ),
      },
      {
        id: 'catatan',
        accessorFn: (row) => row.note ?? '',
        header: labels.note,
        cell: ({ row }) => (
          <span className="text-sm break-words text-[var(--sea-ink-soft)]">
            {row.original.note ?? '-'}
          </span>
        ),
      },
      {
        id: 'sumber',
        accessorFn: (row) => row.source ?? '',
        header: labels.source,
        cell: ({ row }) => (
          <span className="font-mono text-xs break-all text-[var(--sea-ink-soft)]">
            {row.original.source ?? '-'}
          </span>
        ),
      },
    ],
    [labels, t],
  )

  return (
    <div className="space-y-4">
      <div
        className={
          showSearch
            ? 'grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end'
            : 'grid gap-3 sm:max-w-xs'
        }
      >
        {showSearch ? (
          <label className="grid gap-1">
            <span className="text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase">
              {labels.searchLabel}
            </span>
            <div className="relative">
              <Search
                className="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-[var(--sea-ink-soft)]"
                aria-hidden="true"
              />
              <Input
                type="search"
                value={search ?? ''}
                onChange={(event) => onSearchChange?.(event.target.value)}
                placeholder={labels.searchPlaceholder}
                className="h-11 pl-9"
              />
            </div>
          </label>
        ) : null}
        <label className="grid gap-1">
          <span className="text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase">
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
        </label>
      </div>

      <DataTable
        columns={columns}
        data={items}
        getRowId={(row) => row.id}
        sortFields={ATTENDANCE_SORT_FIELDS}
        sort={sort}
        order={order}
        onSortChange={onSortChange}
        page={page}
        pageSize={pageSize}
        totalItems={totalItems}
        totalPages={totalPages}
        onPageChange={onPageChange}
        onPageSizeChange={(size) => onPageSizeChange(size as AttendancePageSize)}
        pageSizeOptions={PAGE_SIZES}
        isFetching={isFetching}
        labels={labels}
      />
    </div>
  )
}
