import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Download } from 'lucide-react'
import { DataTable } from '@/components/DataTable'
import { Input } from '@/components/ui/input'
import { AUDIT_LOG_SORT_FIELDS } from '@/lib/audit'
import type { AuditLogSortField, AuditLogSortOrder } from '@/lib/audit'
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
  sort: AuditLogSortField
  order: AuditLogSortOrder
  onPageChange: (page: number) => void
  onPageSizeChange: (size: AuditLogPageSize) => void
  onSearchChange: (value: string) => void
  onAksiChange: (value: string | null) => void
  onSortChange: (field: AuditLogSortField, order: AuditLogSortOrder) => void
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
    emptyTitle: string
    emptyDescription: string
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
    sortBy: string
    exportCsv: string
  }
}

/**
 * Tabel jejak audit. Backend memegang penyaringan, pengurutan, dan pemotongan
 * baris, jadi tabel ini hanya meneruskan pilihan pengguna ke mesin tabel
 * bersama dan menampilkan baris yang sudah dipotong server.
 */
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
  const columns = useMemo<ColumnDef<AuditLog, unknown>[]>(
    () => [
      {
        id: 'createdAt',
        accessorFn: (row) => row.createdAt,
        header: labels.time,
        cell: ({ row }) => (
          <span className="tabular-nums">
            {new Date(row.original.createdAt).toLocaleString(localeTag)}
          </span>
        ),
      },
      {
        id: 'aksi',
        accessorFn: (row) => row.aksi,
        header: labels.action,
        cell: ({ row }) => (
          <span className="font-medium break-words whitespace-normal">
            {row.original.aksi}
          </span>
        ),
      },
      {
        id: 'aktorId',
        accessorFn: (row) => row.aktorId ?? '',
        header: labels.actor,
        cell: ({ row }) => (
          <span className="text-xs break-all whitespace-normal text-[var(--sea-ink-soft)]">
            {row.original.aktorId ?? labels.noValue}
          </span>
        ),
      },
      {
        id: 'sasaranId',
        accessorFn: (row) => row.sasaranId ?? '',
        header: labels.target,
        cell: ({ row }) => (
          <span className="text-xs break-all whitespace-normal text-[var(--sea-ink-soft)]">
            {row.original.sasaranId ?? labels.noValue}
          </span>
        ),
      },
      {
        id: 'keterangan',
        accessorFn: (row) => row.keterangan ?? '',
        header: labels.note,
        cell: ({ row }) => (
          <span className="break-words whitespace-normal text-[var(--sea-ink-soft)]">
            {row.original.keterangan ?? labels.noValue}
          </span>
        ),
      },
      {
        id: 'ip',
        accessorFn: (row) => row.ip ?? '',
        header: labels.ip,
        cell: ({ row }) => (
          <span className="text-xs whitespace-normal text-[var(--sea-ink-soft)]">
            {row.original.ip ?? labels.noValue}
          </span>
        ),
      },
    ],
    [labels, localeTag],
  )

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-3">
          <label
            htmlFor="audit-search"
            className="text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase"
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
          <label
            htmlFor="audit-action"
            className="text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase"
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

      <DataTable
        columns={columns}
        data={logs}
        getRowId={(row) => row.id}
        sortFields={AUDIT_LOG_SORT_FIELDS}
        sort={sort}
        order={order}
        onSortChange={onSortChange}
        page={page}
        pageSize={pageSize}
        totalItems={totalItems}
        totalPages={totalPages}
        onPageChange={onPageChange}
        onPageSizeChange={(size) => onPageSizeChange(size as AuditLogPageSize)}
        pageSizeOptions={PAGE_SIZES}
        isFetching={isFetching}
        labels={{
          caption: labels.caption,
          loadingLabel: labels.refreshingLabel,
          emptyTitle: labels.emptyTitle,
          emptyDescription: labels.emptyDescription,
          resultLabel: labels.resultLabel,
          pageInfo: labels.pageInfo,
          prevPage: labels.prevPage,
          nextPage: labels.nextPage,
          pageSizeLabel: labels.pageSizeLabel,
          itemsPerPage: labels.itemsPerPage,
          sortBy: labels.sortBy,
        }}
      />
    </div>
  )
}
