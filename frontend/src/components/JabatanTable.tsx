import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Archive, Pencil, Search } from 'lucide-react'
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
import { JABATAN_KINDS, JABATAN_SORT_FIELDS } from '@/lib/jabatan'
import type { JabatanSortField, JabatanSortOrder } from '@/lib/jabatan'
import type { Jabatan } from '@/validations/schemas'

const PAGE_SIZES = [10, 25, 50, 100] as const
export type JabatanPageSize = (typeof PAGE_SIZES)[number]

type JabatanTableLabels = DataTableLabels & {
  name: string
  type: string
  status: string
  active: string
  inactive: string
  searchLabel: string
  searchPlaceholder: string
  jenisLabel: string
  jenisAll: string
  edit: string
  archive: string
  actions: string
  typeStruktural: string
  typeFungsional: string
  emptyHint: string
}

type JabatanTableProps = {
  jabatans: Jabatan[]
  page: number
  pageSize: JabatanPageSize
  totalPages: number
  totalItems: number
  isLoading: boolean
  isFetching: boolean
  search: string
  jenis: string | null
  sort: JabatanSortField
  order: JabatanSortOrder
  onPageChange: (page: number) => void
  onPageSizeChange: (size: JabatanPageSize) => void
  onSearchChange: (value: string) => void
  onSortChange: (field: JabatanSortField, order: JabatanSortOrder) => void
  onJenisChange: (value: string | null) => void
  onEdit: (jabatan: Jabatan) => void
  onArchive: (jabatan: Jabatan) => void
  canManage: boolean
  labels: JabatanTableLabels
}

function jenisLabel(kind: string, labels: JabatanTableLabels): string {
  return kind === 'FUNGSIONAL' ? labels.typeFungsional : labels.typeStruktural
}

/**
 * Tabel daftar master jabatan.
 *
 * Backend memegang pencarian, penyaringan jenis, pengurutan, dan pemotongan
 * baris, jadi tabel ini hanya meneruskan pilihan pengguna.
 */
export default function JabatanTable({
  jabatans,
  page,
  pageSize,
  totalPages,
  totalItems,
  isLoading,
  isFetching,
  search,
  jenis,
  sort,
  order,
  onPageChange,
  onPageSizeChange,
  onSearchChange,
  onSortChange,
  onJenisChange,
  onEdit,
  onArchive,
  canManage,
  labels,
}: JabatanTableProps) {
  const columns = useMemo<ColumnDef<Jabatan, unknown>[]>(() => {
    const base: ColumnDef<Jabatan, unknown>[] = [
      {
        id: 'nama',
        accessorFn: (row) => row.nama,
        header: labels.name,
        cell: ({ row }) => (
          <span className="font-medium break-words text-[var(--sea-ink)]">
            {row.original.nama}
          </span>
        ),
      },
      {
        id: 'jenis',
        accessorFn: (row) => row.jenis,
        header: labels.type,
        cell: ({ row }) => (
          <span className="text-[var(--sea-ink-soft)]">
            {jenisLabel(row.original.jenis, labels)}
          </span>
        ),
      },
      {
        id: 'aktif',
        accessorFn: (row) => (row.aktif ? '1' : '0'),
        header: labels.status,
        cell: ({ row }) => (
          <span
            className={`text-xs font-medium ${
              row.original.aktif ? 'text-[var(--sea-ink)]' : 'text-destructive'
            }`}
          >
            {row.original.aktif ? labels.active : labels.inactive}
          </span>
        ),
      },
    ]

    if (!canManage) {
      return base
    }

    return [
      ...base,
      {
        id: 'aksi',
        enableSorting: false,
        header: () => <span className="sr-only">{labels.actions}</span>,
        cell: ({ row }) => (
          <div className="flex items-center justify-end gap-2">
            <button
              type="button"
              onClick={() => onEdit(row.original)}
              className="inline-flex items-center gap-1 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-1 text-xs font-semibold text-[var(--sea-ink)] transition-colors hover:bg-[var(--surface-strong)]"
            >
              <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
              {labels.edit}
            </button>
            <button
              type="button"
              onClick={() => onArchive(row.original)}
              className="inline-flex items-center gap-1 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-1 text-xs font-semibold text-destructive transition-colors hover:bg-[var(--surface-strong)]"
            >
              <Archive className="h-3.5 w-3.5" aria-hidden="true" />
              {labels.archive}
            </button>
          </div>
        ),
      },
    ]
  }, [labels, canManage, onEdit, onArchive])

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-3">
          <label
            htmlFor="jabatan-search"
            className="text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase"
          >
            {labels.searchLabel}
          </label>
          <div className="relative">
            <Search
              className="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-[var(--sea-ink-soft)]"
              aria-hidden="true"
            />
            <Input
              id="jabatan-search"
              type="search"
              value={search}
              onChange={(event) => onSearchChange(event.target.value)}
              placeholder={labels.searchPlaceholder}
              className="h-11 w-full pl-9 sm:w-64"
            />
          </div>
        </div>

        <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-3">
          <span className="text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase">
            {labels.jenisLabel}
          </span>
          <Select
            value={jenis ?? 'all'}
            onValueChange={(value) => onJenisChange(value === 'all' ? null : value)}
          >
            <SelectTrigger className="h-11 min-w-[12rem]" aria-label={labels.jenisLabel}>
              <SelectValue placeholder={labels.jenisAll} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{labels.jenisAll}</SelectItem>
              {JABATAN_KINDS.map((option) => (
                <SelectItem key={option} value={option}>
                  {jenisLabel(option, labels)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={jabatans}
        getRowId={(row) => row.id}
        sortFields={JABATAN_SORT_FIELDS}
        sort={sort}
        order={order}
        onSortChange={onSortChange}
        page={page}
        pageSize={pageSize}
        totalItems={totalItems}
        totalPages={totalPages}
        onPageChange={onPageChange}
        onPageSizeChange={(size) => onPageSizeChange(size as JabatanPageSize)}
        pageSizeOptions={PAGE_SIZES}
        isLoading={isLoading}
        isFetching={isFetching}
        emptyState={
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">{labels.emptyHint}</p>
        }
        labels={labels}
      />
    </div>
  )
}
