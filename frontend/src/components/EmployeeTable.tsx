import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Search } from 'lucide-react'
import { DataTable } from '@/components/DataTable'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { EMPLOYEE_SORT_FIELDS } from '@/lib/employees'
import type { EmployeeSortField, EmployeeSortOrder } from '@/lib/employees'
import type { Employee } from '@/validations/schemas'

const PAGE_SIZES = [10, 20, 25, 50, 100] as const
export type EmployeePageSize = (typeof PAGE_SIZES)[number]

type EmployeeTableProps = {
  employees: Employee[]
  unitNames: Map<string, string>
  sort: EmployeeSortField
  order: EmployeeSortOrder
  page: number
  pageSize: EmployeePageSize
  totalPages: number
  totalItems: number
  isFetching: boolean
  search: string
  homebaseId: string | null
  onSortChange: (field: EmployeeSortField, order: EmployeeSortOrder) => void
  onPageChange: (page: number) => void
  onPageSizeChange: (size: EmployeePageSize) => void
  onSearchChange: (value: string) => void
  onHomebaseChange: (value: string | null) => void
  labels: {
    caption: string
    name: string
    nip: string
    unit: string
    position: string
    status: string
    active: string
    inactive: string
    noUnit: string
    noValue: string
    resultLabel: string
    refreshingLabel: string
    prevPage: string
    nextPage: string
    pageInfo: string
    searchLabel: string
    searchPlaceholder: string
    unitLabel: string
    unitAll: string
    pageSizeLabel: string
    itemsPerPage: string
    sortBy: string
  }
}

/**
 * Tabel daftar pegawai: pencarian, filter unit, pengurutan, ukuran halaman,
 * dan paginasi. Urutan, penyaringan, dan pemotongan baris dikerjakan backend,
 * jadi tabel hanya mengirim pilihan pengguna.
 */
export default function EmployeeTable({
  employees,
  unitNames,
  sort,
  order,
  page,
  pageSize,
  totalPages,
  totalItems,
  isFetching,
  search,
  homebaseId,
  onSortChange,
  onPageChange,
  onPageSizeChange,
  onSearchChange,
  onHomebaseChange,
  labels,
}: EmployeeTableProps) {
  const columns = useMemo<ColumnDef<Employee, unknown>[]>(
    () => [
      {
        id: 'nama',
        accessorFn: (row) => row.nama,
        header: labels.name,
        cell: ({ row }) => (
          <span className="block">
            <span className="block font-semibold break-words">{row.original.nama}</span>
            <span className="block text-xs break-all text-[var(--sea-ink-soft)]">
              {row.original.email}
            </span>
          </span>
        ),
      },
      {
        id: 'nip',
        accessorFn: (row) => row.nip ?? '',
        header: labels.nip,
        cell: ({ row }) =>
          row.original.nip ? (
            <span className="font-mono text-xs">{row.original.nip}</span>
          ) : (
            <span className="text-[var(--sea-ink-soft)]">{labels.noValue}</span>
          ),
      },
      {
        id: 'homebaseId',
        accessorFn: (row) => row.homebaseId ?? '',
        header: labels.unit,
        cell: ({ row }) =>
          row.original.homebaseId ? (
            <span className="break-words">
              {unitNames.get(row.original.homebaseId) ?? row.original.homebaseId}
            </span>
          ) : (
            <span className="text-[var(--sea-ink-soft)]">{labels.noUnit}</span>
          ),
      },
      {
        id: 'jabatanStruktural',
        accessorFn: (row) => row.jabatanStruktural ?? row.jabatanFungsional ?? '',
        header: labels.position,
        cell: ({ row }) =>
          row.original.jabatanStruktural ??
          row.original.jabatanFungsional ?? (
            <span className="text-[var(--sea-ink-soft)]">{labels.noValue}</span>
          ),
      },
      {
        id: 'aktif',
        accessorFn: (row) => (row.aktif ? '1' : '0'),
        header: labels.status,
        cell: ({ row }) => (
          <span
            className={
              row.original.aktif
                ? 'font-medium text-[var(--sea-ink)]'
                : 'text-[var(--sea-ink-soft)]'
            }
          >
            {row.original.aktif ? labels.active : labels.inactive}
          </span>
        ),
      },
    ],
    [labels, unitNames],
  )

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-3">
          <label
            htmlFor="employee-search"
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
              id="employee-search"
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
            {labels.unitLabel}
          </span>
          <Select
            value={homebaseId ?? 'all'}
            onValueChange={(value) => onHomebaseChange(value === 'all' ? null : value)}
          >
            <SelectTrigger className="h-11 min-w-[14rem]" aria-label={labels.unitLabel}>
              <SelectValue placeholder={labels.unitAll} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{labels.unitAll}</SelectItem>
              {[...unitNames.entries()].map(([id, name]) => (
                <SelectItem key={id} value={id}>
                  {name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={employees}
        sortFields={EMPLOYEE_SORT_FIELDS}
        sort={sort}
        order={order}
        onSortChange={onSortChange}
        page={page}
        pageSize={pageSize}
        totalItems={totalItems}
        totalPages={totalPages}
        onPageChange={onPageChange}
        onPageSizeChange={(size) => onPageSizeChange(size as EmployeePageSize)}
        pageSizeOptions={PAGE_SIZES}
        isFetching={isFetching}
        labels={{
          caption: labels.caption,
          loadingLabel: labels.refreshingLabel,
          emptyTitle: labels.caption,
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
