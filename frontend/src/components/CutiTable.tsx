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
import { CUTI_PAGE_SIZES } from '@/lib/cuti'
import type { Cuti } from '@/validations/schemas'

export type CutiPageSize = (typeof CUTI_PAGE_SIZES)[number]

type CutiTableProps = {
  cutis: Cuti[]
  page: number
  pageSize: CutiPageSize
  totalPages: number
  totalItems: number
  search: string
  jenis: string | null
  status: string | null
  sort: CutiSortField
  order: 'asc' | 'desc'
  onPageChange: (page: number) => void
  onSearchChange: (value: string) => void
  onSortChange: (field: CutiSortField) => void
  onJenisChange: (value: string | null) => void
  onStatusChange: (value: string | null) => void
  onApprove: (cuti: Cuti) => void
  onReject: (cuti: Cuti) => void
  canApprove: boolean
  canReject: boolean
  labels: {
    caption: string
    user: string
    jenis: string
    tanggalMulai: string
    tanggalSelesai: string
    status: string
    createdAt: string
    resultLabel: string
    refreshingLabel: string
    prevPage: string
    nextPage: string
    pageInfo: string
    searchLabel: string
    searchPlaceholder: string
    jenisLabel: string
    jenisAll: string
    statusLabel: string
    statusAll: string
    pageSizeLabel: string
    itemsPerPage: string
    approve: string
    reject: string
    statusMenunggu: string
    statusDisetujui: string
    statusDitolak: string
  }
}

export type CutiSortField =
  | 'userId'
  | 'jenis'
  | 'tanggalMulai'
  | 'tanggalSelesai'
  | 'status'
  | 'createdAt'
  | 'updatedAt'

function pageRange(totalItems: number, page: number, pageSize: number): [number, number] {
  if (totalItems === 0) {
    return [0, 0]
  }
  const from = (page - 1) * pageSize + 1
  const to = Math.min(page * pageSize, totalItems)
  return [from, to]
}

function formatDateISO(value: string): string {
  return new Date(value).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function statusBadge(status: string): string {
  switch (status) {
    case 'MENUNGGU':
      return 'bg-[var(--chip-bg)] border border-[var(--chip-line)] text-[var(--sea-ink)]'
    case 'DISETUJUI':
      return 'bg-[var(--palm-bg)] text-[var(--palm)]'
    case 'DITOLAK':
      return 'bg-red-50 text-red-600 border border-red-200'
    default:
      return 'bg-[var(--chip-bg)] border border-[var(--chip-line)] text-[var(--sea-ink)]'
  }
}

function statusLabel(status: string, labels: CutiTableProps['labels']): string {
  switch (status) {
    case 'MENUNGGU':
      return labels.statusMenunggu
    case 'DISETUJUI':
      return labels.statusDisetujui
    case 'DITOLAK':
      return labels.statusDitolak
    default:
      return status
  }
}

const JENIS_OPTIONS: { value: string; label: string }[] = [
  { value: 'TAHUNAN', label: 'Tahunan' },
  { value: 'PANJANG', label: 'Panjang' },
  { value: 'BERSALIN', label: 'Bersalin' },
  { value: 'SAKIT', label: 'Sakit' },
  { value: 'SAKIT_BERKEPANJANGAN', label: 'Sakit Berkepanjangan' },
  { value: 'IZIN_PENTING', label: 'Izin Penting' },
  { value: 'POLITIK', label: 'Politik' },
]

const STATUS_OPTIONS: { value: string; label: string }[] = [
  { value: 'MENUNGGU', label: 'Menunggu' },
  { value: 'DISETUJUI', label: 'Disetujui' },
  { value: 'DITOLAK', label: 'Ditolak' },
]

export default function CutiTable({
  cutis,
  page,
  pageSize,
  totalPages,
  totalItems,
  search,
  jenis,
  status,
  sort,
  order,
  onPageChange,
  onSearchChange,
  onSortChange,
  onJenisChange,
  onStatusChange,
  onApprove,
  onReject,
  canApprove,
  canReject,
  labels,
}: CutiTableProps) {
  const [from, to] = pageRange(totalItems, page, pageSize)

  const isActive = (field: CutiSortField) => sort === field

  return (
    <div className="space-y-4">
      <TableCaption className="mb-2 text-sm font-semibold text-[var(--sea-ink)]">
        {labels.caption}
      </TableCaption>
      <div className="rounded-xl border border-[var(--line)] bg-[var(--chip-bg)] p-3">
        <div className="grid gap-2 md:grid-cols-3 items-end">
          <div>
            <label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)] mb-1 block">
              {labels.searchLabel}
            </label>
            <Input
              value={search}
              onChange={(event) => onSearchChange(event.target.value)}
              placeholder={labels.searchPlaceholder}
              className="h-9"
            />
          </div>
          <div>
            <label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)] mb-1 block">
              {labels.jenisLabel}
            </label>
            <Select value={jenis ?? ''} onValueChange={onJenisChange}>
              <SelectTrigger className="h-9">
                <SelectValue placeholder={labels.jenisAll} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value=""> {labels.jenisAll} </SelectItem>
                {JENIS_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)] mb-1 block">
              {labels.statusLabel}
            </label>
            <Select value={status ?? ''} onValueChange={onStatusChange}>
              <SelectTrigger className="h-9">
                <SelectValue placeholder={labels.statusAll} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value=""> {labels.statusAll} </SelectItem>
                {STATUS_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      <Table>
        <TableHeader>
          <TableRow className="text-xs text-[var(--sea-ink-soft)]">
            <TableHead className="w-24">{labels.user}</TableHead>
            <TableHead className="w-20">{labels.jenis}</TableHead>
            <TableHead className="w-20">
              <button
                onClick={() => onSortChange('tanggalMulai')}
                className="flex items-center gap-1 text-[var(--sea-ink)] hover:text-[var(--lagoon)]"
              >
                {labels.tanggalMulai}
                <SortIcon active={isActive('tanggalMulai')} order={order} />
              </button>
            </TableHead>
            <TableHead className="w-20">
              <button
                onClick={() => onSortChange('tanggalSelesai')}
                className="flex items-center gap-1 text-[var(--sea-ink)] hover:text-[var(--lagoon)]"
              >
                {labels.tanggalSelesai}
                <SortIcon active={isActive('tanggalSelesai')} order={order} />
              </button>
            </TableHead>
            <TableHead className="w-16">
              <button
                onClick={() => onSortChange('status')}
                className="flex items-center gap-1 text-[var(--sea-ink)] hover:text-[var(--lagoon)]"
              >
                {labels.status}
                <SortIcon active={isActive('status')} order={order} />
              </button>
            </TableHead>
            <TableHead className="w-20">{labels.createdAt}</TableHead>
            <TableHead className="w-24 text-right" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {cutis.map((cuti) => (
            <TableRow key={cuti.id}>
              <TableCell className="w-24 font-medium text-[var(--sea-ink)]">
                {cuti.userId}
              </TableCell>
              <TableCell className="w-20 text-[var(--sea-ink)]">
                {JENIS_OPTIONS.find((o) => o.value === cuti.jenis)?.label ?? cuti.jenis}
              </TableCell>
              <TableCell className="w-20 text-[var(--sea-ink)]">
                {formatDateISO(cuti.tanggalMulai)}
              </TableCell>
              <TableCell className="w-20 text-[var(--sea-ink)]">
                {formatDateISO(cuti.tanggalSelesai)}
              </TableCell>
              <TableCell className="w-16">
                <span
                  className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${statusBadge(cuti.status)}`}
                >
                  {statusLabel(cuti.status, labels)}
                </span>
              </TableCell>
              <TableCell className="w-20 text-[var(--sea-ink)]">
                {formatDateISO(cuti.createdAt)}
              </TableCell>
              <TableCell className="w-24 text-right">
                {cuti.status === 'MENUNGGU' && canApprove ? (
                  <>
                    <button
                      type="button"
                      onClick={() => onApprove(cuti)}
                      className="inline-flex items-center gap-1 rounded-md border border-[var(--chip-line)] bg-[var(--chip-bg)] px-2 py-1 text-xs font-medium text-[var(--sea-ink)] hover:bg-[var(--link-bg-hover)]"
                    >
                      {labels.approve}
                    </button>
                    {canReject ? (
                      <button
                        type="button"
                        onClick={() => onReject(cuti)}
                        className="ml-2 inline-flex items-center gap-1 rounded-md border border-[var(--chip-line)] bg-[var(--chip-bg)] px-2 py-1 text-xs font-medium text-[var(--sea-ink)] hover:bg-[var(--link-bg-hover)]"
                      >
                        {labels.reject}
                      </button>
                    ) : null}
                  </>
                ) : null}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <div className="flex items-center justify-between gap-2 sm:justify-end">
        <button
          type="button"
          onClick={() => onPageChange(page - 1)}
          disabled={page === 1}
          className="inline-flex items-center rounded-md border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-1.5 text-xs font-medium text-[var(--sea-ink)] disabled:opacity-50 disabled:pointer-events-none hover:bg-[var(--link-bg-hover)]"
        >
          <ChevronLeft className="h-4 w-4" aria-hidden="true" />
          {labels.prevPage}
        </button>
        <span className="text-xs font-medium text-[var(--sea-ink)]">
          {labels.pageInfo
            .replace('{page}', String(page))
            .replace('{total}', String(totalPages))}
        </span>
        <button
          type="button"
          onClick={() => onPageChange(page + 1)}
          disabled={page === totalPages}
          className="inline-flex items-center rounded-md border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-1.5 text-xs font-medium text-[var(--sea-ink)] disabled:opacity-50 disabled:pointer-events-none hover:bg-[var(--link-bg-hover)]"
        >
          {labels.nextPage}
          <ChevronRight className="h-4 w-4" aria-hidden="true" />
        </button>
      </div>

      <p className="m-0 text-xs text-[var(--sea-ink-soft)]">
        {labels.itemsPerPage
          .replace('{from}', String(from))
          .replace('{to}', String(to))
          .replace('{total}', String(totalItems))}
      </p>
    </div>
  )
}
