import { ChevronLeft, ChevronRight, Download } from 'lucide-react'
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
import type { AuditLog } from '@/validations/schemas'

const PAGE_SIZES = [10, 25, 50, 100] as const
export type AuditLogPageSize = (typeof PAGE_SIZES)[number]

type AuditLogTableProps = {
  logs: AuditLog[]
  page: number
  pageSize: AuditLogPageSize
  totalPages: number
  totalItems: number
  isFetching: boolean
  isExporting: boolean
  search: string
  aksi: string | null
  sort: 'createdAt' | 'aksi'
  order: 'asc' | 'desc'
  onPageChange: (page: number) => void
  onPageSizeChange: (size: AuditLogPageSize) => void
  onSearchChange: (value: string) => void
  onAksiChange: (value: string | null) => void
  onSortChange: (field: 'createdAt' | 'aksi') => void
  onExport: () => void
  localeTag: string
  labels: {
    caption: string
    time: string
    action: string
    actor: string
    target: string
    note: string
    ip: string
    noValue: string
    resultLabel: string
    refreshingLabel: string
    prevPage: string
    nextPage: string
    pageInfo: string
    searchLabel: string
    searchPlaceholder: string
    actionLabel: string
    actionPlaceholder: string
    pageSizeLabel: string
    itemsPerPage: string
    exportCsv: string
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

export default function AuditLogTable({
  logs,
  page,
  pageSize,
  totalPages,
  totalItems,
  isFetching,
  isExporting,
  search,
  aksi,
  sort,
  order,
  onPageChange,
  onPageSizeChange,
  onSearchChange,
  onAksiChange,
  onSortChange,
  onExport,
  localeTag,
  labels,
}: AuditLogTableProps) {
  const [from, to] = pageRange(totalItems, page, pageSize)

  return (
    <div className="island-shell overflow-hidden rounded-2xl">
      <div className="border-b border-[var(--line)] px-4 py-3">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:gap-4">
            <div className="flex flex-col gap-1">
              <label
                htmlFor="audit-search"
                className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
              >
                {labels.searchLabel}
              </label>
              <Input
                id="audit-search"
                type="search"
                value={search}
                onChange={(event) => onSearchChange(event.target.value)}
                placeholder={labels.searchPlaceholder}
                className="h-11 w-full sm:w-64"
              />
            </div>
            <div className="flex flex-col gap-1">
              <label
                htmlFor="audit-action"
                className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
              >
                {labels.actionLabel}
              </label>
              <Input
                id="audit-action"
                type="text"
                value={aksi ?? ''}
                onChange={(event) => onAksiChange(event.target.value.trim() || null)}
                placeholder={labels.actionPlaceholder}
                className="h-11 w-full sm:w-56"
              />
            </div>
          </div>

          <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:gap-4">
            <div className="flex flex-col gap-1">
              <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
                {labels.pageSizeLabel}
              </span>
              <Select
                value={String(pageSize)}
                onValueChange={(value) =>
                  onPageSizeChange(Number(value) as AuditLogPageSize)
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
            <button
              type="button"
              onClick={onExport}
              disabled={isExporting}
              className="inline-flex h-11 items-center gap-2 self-start rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-4 text-xs font-semibold text-[var(--sea-ink)] transition hover:bg-[var(--link-bg-hover)] disabled:cursor-not-allowed disabled:opacity-60"
            >
              <Download className="h-4 w-4" aria-hidden="true" />
              {labels.exportCsv}
            </button>
          </div>
        </div>
      </div>

      <Table className="min-w-[48rem]">
        <TableCaption className="sr-only">{labels.caption}</TableCaption>
        <TableHeader>
          <TableRow className="border-[var(--line)] hover:bg-transparent">
            <TableHead scope="col" className="px-4 py-3">
              <button
                type="button"
                onClick={() => onSortChange('createdAt')}
                aria-label={labels.time}
                className="island-kicker inline-flex items-center gap-1"
              >
                {labels.time}
                <SortIcon active={sort === 'createdAt'} order={order} />
              </button>
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              <button
                type="button"
                onClick={() => onSortChange('aksi')}
                aria-label={labels.action}
                className="island-kicker inline-flex items-center gap-1"
              >
                {labels.action}
                <SortIcon active={sort === 'aksi'} order={order} />
              </button>
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.actor}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.target}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.note}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.ip}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody aria-busy={isFetching}>
          {logs.map((log) => (
            <TableRow
              key={log.id}
              className="border-[var(--line)] last:border-0 align-top"
            >
              <TableCell className="px-4 py-3 tabular-nums text-[var(--sea-ink)]">
                {new Date(log.createdAt).toLocaleString(localeTag)}
              </TableCell>
              <TableCell className="px-4 py-3 font-medium text-[var(--sea-ink)] break-words">
                {log.aksi}
              </TableCell>
              <TableCell className="px-4 py-3 text-[var(--sea-ink-soft)] break-all text-xs">
                {log.aktorId ?? labels.noValue}
              </TableCell>
              <TableCell className="px-4 py-3 text-[var(--sea-ink-soft)] break-all text-xs">
                {log.sasaranId ?? labels.noValue}
              </TableCell>
              <TableCell className="px-4 py-3 text-[var(--sea-ink-soft)] break-words">
                {log.keterangan ?? labels.noValue}
              </TableCell>
              <TableCell className="px-4 py-3 text-[var(--sea-ink-soft)] text-xs">
                {log.ip ?? labels.noValue}
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
          {[...Array(Math.max(totalPages, 1))].map((_, index) => {
            const n = index + 1
            const active = n === page
            return (
              <button
                key={n}
                type="button"
                onClick={() => onPageChange(n)}
                aria-current={active ? 'page' : undefined}
                className={
                  active
                    ? 'h-10 w-10 rounded-full bg-[var(--lagoon)] text-white text-xs font-semibold'
                    : 'h-10 w-10 rounded-full border border-[var(--line)] bg-[var(--chip-bg)] text-[var(--sea-ink)] text-xs font-semibold hover:bg-[var(--surface-strong)]'
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
