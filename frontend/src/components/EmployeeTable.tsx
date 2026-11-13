import type { Employee } from '@/validations/schemas'
import type { EmployeeSortField, EmployeeSortOrder } from '@/lib/employees'
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
  onSortChange: (field: EmployeeSortField) => void
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

/**
 * Tabel daftar pegawai dengan kontrol lengkap: pencarian, filter unit,
 * pengurutan, ukuran halaman (10/25/50/100), dan paginasi. Urutan dan
 * penomoran halaman dikendalikan backend.
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
  const [from, to] = pageRange(totalItems, page, pageSize)

  return (
    <div className="island-shell overflow-hidden rounded-2xl">
      {/* Kontrol atas: pencarian, filter unit, ukuran halaman. */}
      <div className="border-b border-[var(--line)] px-4 py-3 sm:flex sm:items-center sm:justify-between gap-3">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center">
          <label
            htmlFor="employee-search"
            className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
          >
            {labels.searchLabel}
          </label>
          <Input
            id="employee-search"
            type="search"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder={labels.searchPlaceholder}
            className="h-11 w-full sm:w-64"
          />
        </div>

        <div className="flex flex-col gap-1 sm:flex-row sm:items-center">
          <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
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

        <div className="flex flex-col gap-1 sm:flex-row sm:items-center">
          <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
            {labels.pageSizeLabel}
          </span>
          <Select
            value={String(pageSize)}
            onValueChange={(value) =>
              onPageSizeChange(value as unknown as EmployeePageSize)
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

      <Table className="min-w-[42rem]">
        <TableCaption className="sr-only">{labels.caption}</TableCaption>
        <TableHeader>
          <TableRow className="border-[var(--line)] hover:bg-transparent">
            <TableHead
              scope="col"
              className="sticky left-0 z-10 border-r border-[var(--line)] bg-[var(--card)] px-4 py-3"
            >
              <button
                type="button"
                onClick={() => onSortChange('nama')}
                aria-label={labels.name}
                className="island-kicker -mx-2 inline-flex min-h-11 items-center gap-1 px-2 sm:min-h-0"
              >
                {labels.name}
                <SortIcon active={sort === 'nama'} order={order} />
              </button>
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.nip}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.unit}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.position}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.status}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody aria-busy={isFetching}>
          {employees.map((employee) => (
            <TableRow key={employee.id} className="border-[var(--line)] last:border-0">
              <TableHead
                scope="row"
                className="sticky left-0 z-10 border-r border-[var(--line)] bg-[var(--card)] px-4 py-3 text-left font-semibold whitespace-normal"
              >
                <span className="block break-words">{employee.nama}</span>
                <span className="block break-all text-xs font-normal text-[var(--sea-ink-soft)]">
                  {employee.email}
                </span>
              </TableHead>
              <TableCell className="px-4 py-3 font-mono text-xs">
                {employee.nip ?? (
                  <span className="font-sans text-[var(--sea-ink-soft)]">
                    {labels.noValue}
                  </span>
                )}
              </TableCell>
              <TableCell className="px-4 py-3">
                {employee.homebaseId ? (
                  <span className="break-words">
                    {unitNames.get(employee.homebaseId) ?? employee.homebaseId}
                  </span>
                ) : (
                  <span className="text-[var(--sea-ink-soft)]">{labels.noUnit}</span>
                )}
              </TableCell>
              <TableCell className="px-4 py-3">
                {employee.jabatanStruktural ?? employee.jabatanFungsional ?? (
                  <span className="text-[var(--sea-ink-soft)]">{labels.noValue}</span>
                )}
              </TableCell>
              <TableCell className="px-4 py-3">
                {employee.aktif ? labels.active : labels.inactive}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {/* Paginasi: prev, angka halaman, next. */}
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
                className={cn(
                  'h-10 w-10 rounded-full text-xs font-semibold transition',
                  active
                    ? 'bg-[var(--lagoon)] text-white shadow-[0_0_0_1px_var(--lagoon)_inset]'
                    : 'border border-[var(--line)] bg-[var(--chip-bg)] text-[var(--sea-ink)] hover:bg-[var(--surface-strong)] disabled:opacity-50',
                )}
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

function cn(...classes: (string | boolean | undefined | null)[]): string {
  return classes.filter(Boolean).join(' ')
}
