import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { DataTable } from '@/components/DataTable'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { CUTI_PAGE_SIZES, CUTI_SORT_FIELDS } from '@/lib/cuti'
import type { CutiSortField, CutiSortOrder } from '@/lib/cuti'
import type { Cuti } from '@/validations/schemas'

export type CutiPageSize = (typeof CUTI_PAGE_SIZES)[number]

type CutiTableProps = {
  cutis: Cuti[]
  page: number
  pageSize: CutiPageSize
  totalPages: number
  totalItems: number
  isFetching: boolean
  search: string
  jenis: string | null
  status: string | null
  sort: CutiSortField
  order: CutiSortOrder
  onPageChange: (page: number) => void
  onPageSizeChange: (size: CutiPageSize) => void
  onSearchChange: (value: string) => void
  onSortChange: (field: CutiSortField, order: CutiSortOrder) => void
  onJenisChange: (value: string | null) => void
  onStatusChange: (value: string | null) => void
  onApprove: (cuti: Cuti) => void
  onReject: (cuti: Cuti) => void
  onCancel: (cuti: Cuti) => void
  canApprove: boolean
  canReject: boolean
  canCancelSelf: boolean
  labels: {
    caption: string
    user: string
    jenis: string
    tanggalMulai: string
    tanggalSelesai: string
    status: string
    createdAt: string
    action: string
    emptyTitle: string
    emptyDescription: string
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
    sortBy: string
    approve: string
    reject: string
    cancel: string
    statusMenunggu: string
    statusDisetujui: string
    statusDitolak: string
    statusBatal: string
    jenisLabels: Record<string, string>
  }
}

const JENIS_VALUES = [
  'TAHUNAN',
  'PANJANG',
  'BERSALIN',
  'SAKIT',
  'SAKIT_BERKEPANJANGAN',
  'IZIN_PENTING',
  'POLITIK',
] as const

const STATUS_VALUES = ['MENUNGGU', 'DISETUJUI', 'DITOLAK', 'BATAL'] as const

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
    case 'BATAL':
      return labels.statusBatal
    default:
      return status
  }
}

/**
 * Tabel daftar pengajuan cuti. Backend memegang penyaringan, pengurutan, dan
 * pemotongan baris, jadi tabel ini hanya meneruskan pilihan pengguna ke mesin
 * tabel bersama.
 */
export default function CutiTable({
  cutis,
  page,
  pageSize,
  totalPages,
  totalItems,
  isFetching,
  search,
  jenis,
  status,
  sort,
  order,
  onPageChange,
  onPageSizeChange,
  onSearchChange,
  onSortChange,
  onJenisChange,
  onStatusChange,
  onApprove,
  onReject,
  canApprove,
  canReject,
  canCancelSelf,
  onCancel,
  labels,
}: CutiTableProps) {
  const columns = useMemo<ColumnDef<Cuti, unknown>[]>(
    () => [
      {
        id: 'userId',
        accessorFn: (row) => row.userId,
        header: labels.user,
        cell: ({ row }) => (
          <span className="font-medium whitespace-normal break-all text-[var(--sea-ink)]">
            {row.original.userId}
          </span>
        ),
      },
      {
        id: 'jenis',
        accessorFn: (row) => row.jenis,
        header: labels.jenis,
        cell: ({ row }) => (
          <span className="whitespace-normal text-[var(--sea-ink)]">
            {labels.jenisLabels[row.original.jenis] ?? row.original.jenis}
          </span>
        ),
      },
      {
        id: 'tanggalMulai',
        accessorFn: (row) => row.tanggalMulai,
        header: labels.tanggalMulai,
        cell: ({ row }) => (
          <span className="whitespace-normal text-[var(--sea-ink)]">
            {formatDateISO(row.original.tanggalMulai)}
          </span>
        ),
      },
      {
        id: 'tanggalSelesai',
        accessorFn: (row) => row.tanggalSelesai,
        header: labels.tanggalSelesai,
        cell: ({ row }) => (
          <span className="whitespace-normal text-[var(--sea-ink)]">
            {formatDateISO(row.original.tanggalSelesai)}
          </span>
        ),
      },
      {
        id: 'status',
        accessorFn: (row) => row.status,
        header: labels.status,
        cell: ({ row }) => (
          <span
            className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${statusBadge(row.original.status)}`}
          >
            {statusLabel(row.original.status, labels)}
          </span>
        ),
      },
      {
        id: 'createdAt',
        accessorFn: (row) => row.createdAt,
        header: labels.createdAt,
        cell: ({ row }) => (
          <span className="whitespace-normal text-[var(--sea-ink)]">
            {formatDateISO(row.original.createdAt)}
          </span>
        ),
      },
      {
        id: 'aksi',
        header: labels.action,
        cell: ({ row }) =>
          (row.original.status === 'MENUNGGU' || row.original.status === 'DISETUJUI') &&
          (canApprove || canCancelSelf) ? (
            <div className="flex justify-end">
              {row.original.status === 'MENUNGGU' ? (
                <button
                  type="button"
                  onClick={() => onApprove(row.original)}
                  className="inline-flex items-center gap-1 rounded-md border border-[var(--chip-line)] bg-[var(--chip-bg)] px-2 py-1 text-xs font-medium text-[var(--sea-ink)] hover:bg-[var(--link-bg-hover)]"
                >
                  {labels.approve}
                </button>
              ) : null}
              {row.original.status === 'MENUNGGU' && canReject ? (
                <button
                  type="button"
                  onClick={() => onReject(row.original)}
                  className="ml-2 inline-flex items-center gap-1 rounded-md border border-[var(--chip-line)] bg-[var(--chip-bg)] px-2 py-1 text-xs font-medium text-[var(--sea-ink)] hover:bg-[var(--link-bg-hover)]"
                >
                  {labels.reject}
                </button>
              ) : null}
              <button
                type="button"
                onClick={() => onCancel(row.original)}
                className="ml-2 inline-flex items-center gap-1 rounded-md border border-[var(--chip-line)] bg-[var(--chip-bg)] px-2 py-1 text-xs font-medium text-destructive hover:bg-[var(--link-bg-hover)]"
              >
                {labels.cancel}
              </button>
            </div>
          ) : null,
      },
    ],
    [labels, canApprove, canReject, canCancelSelf, onApprove, onReject, onCancel],
  )

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-[var(--line)] bg-[var(--chip-bg)] p-3">
        <div className="grid gap-2 md:grid-cols-3 md:items-end">
          <div>
            <label
              htmlFor="cuti-search"
              className="mb-1 block text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase"
            >
              {labels.searchLabel}
            </label>
            <Input
              id="cuti-search"
              type="search"
              value={search}
              onChange={(event) => onSearchChange(event.target.value)}
              placeholder={labels.searchPlaceholder}
              className="h-9"
            />
          </div>
          <div>
            <label
              htmlFor="cuti-jenis"
              className="mb-1 block text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase"
            >
              {labels.jenisLabel}
            </label>
            <Select
              value={jenis ?? ''}
              onValueChange={(value) => onJenisChange(value || null)}
            >
              <SelectTrigger id="cuti-jenis" className="h-9">
                <SelectValue placeholder={labels.jenisAll} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">{labels.jenisAll}</SelectItem>
                {JENIS_VALUES.map((value) => (
                  <SelectItem key={value} value={value}>
                    {labels.jenisLabels[value] ?? value}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <label
              htmlFor="cuti-status"
              className="mb-1 block text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase"
            >
              {labels.statusLabel}
            </label>
            <Select
              value={status ?? ''}
              onValueChange={(value) => onStatusChange(value || null)}
            >
              <SelectTrigger id="cuti-status" className="h-9">
                <SelectValue placeholder={labels.statusAll} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">{labels.statusAll}</SelectItem>
                {STATUS_VALUES.map((value) => (
                  <SelectItem key={value} value={value}>
                    {statusLabel(value, labels)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={cutis}
        getRowId={(row) => row.id}
        sortFields={CUTI_SORT_FIELDS}
        sort={sort}
        order={order}
        onSortChange={onSortChange}
        page={page}
        pageSize={pageSize}
        totalItems={totalItems}
        totalPages={totalPages}
        onPageChange={onPageChange}
        onPageSizeChange={(size) => onPageSizeChange(size as CutiPageSize)}
        pageSizeOptions={CUTI_PAGE_SIZES}
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
