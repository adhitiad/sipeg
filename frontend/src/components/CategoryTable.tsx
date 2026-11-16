import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Search } from 'lucide-react'
import { DataTable } from '@/components/DataTable'
import type { DataTableLabels } from '@/components/DataTable'
import { Input } from '@/components/ui/input'
import { CATEGORY_SORT_FIELDS } from '@/lib/audit'
import type { CategorySortField, CategorySortOrder } from '@/lib/audit'
import type { Category } from '@/validations/schemas'

const PAGE_SIZES = [10, 25, 50, 100] as const
export type CategoryPageSize = (typeof PAGE_SIZES)[number]

type CategoryTableLabels = DataTableLabels & {
  code: string
  name: string
  retirementAge: string
  status: string
  active: string
  inactive: string
  searchLabel: string
  searchPlaceholder: string
  emptyHint: string
}

type CategoryTableProps = {
  categories: Category[]
  page: number
  pageSize: CategoryPageSize
  totalPages: number
  totalItems: number
  isLoading: boolean
  isFetching: boolean
  search: string
  sort: CategorySortField
  order: CategorySortOrder
  onPageChange: (page: number) => void
  onPageSizeChange: (size: CategoryPageSize) => void
  onSearchChange: (value: string) => void
  onSortChange: (field: CategorySortField, order: CategorySortOrder) => void
  labels: CategoryTableLabels
}

/**
 * Tabel daftar kategori pegawai.
 *
 * Backend memegang pencarian, pengurutan, dan pemotongan baris, jadi tabel ini
 * hanya meneruskan pilihan pengguna tanpa memfilter baris sendiri.
 */
export default function CategoryTable({
  categories,
  page,
  pageSize,
  totalPages,
  totalItems,
  isLoading,
  isFetching,
  search,
  sort,
  order,
  onPageChange,
  onPageSizeChange,
  onSearchChange,
  onSortChange,
  labels,
}: CategoryTableProps) {
  const columns = useMemo<ColumnDef<Category, unknown>[]>(
    () => [
      {
        id: 'kode',
        accessorFn: (row) => row.kode,
        header: labels.code,
        cell: ({ row }) => (
          <span className="font-medium break-words text-[var(--sea-ink)]">
            {row.original.kode}
          </span>
        ),
      },
      {
        id: 'nama',
        accessorFn: (row) => row.nama,
        header: labels.name,
        cell: ({ row }) => <span className="break-words">{row.original.nama}</span>,
      },
      {
        id: 'usiaPensiun',
        accessorFn: (row) => row.usiaPensiun,
        header: labels.retirementAge,
        cell: ({ row }) => (
          <span className="tabular-nums text-[var(--sea-ink-soft)]">
            {row.original.usiaPensiun}
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
    ],
    [labels],
  )

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-3">
        <label
          htmlFor="category-search"
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
            id="category-search"
            type="search"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder={labels.searchPlaceholder}
            className="h-11 w-full pl-9 sm:w-64"
          />
        </div>
      </div>

      <DataTable
        columns={columns}
        data={categories}
        getRowId={(row) => row.id}
        sortFields={CATEGORY_SORT_FIELDS}
        sort={sort}
        order={order}
        onSortChange={onSortChange}
        page={page}
        pageSize={pageSize}
        totalItems={totalItems}
        totalPages={totalPages}
        onPageChange={onPageChange}
        onPageSizeChange={(size) => onPageSizeChange(size as CategoryPageSize)}
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
