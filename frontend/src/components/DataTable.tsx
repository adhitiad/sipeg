import { useMemo } from 'react'
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
  type OnChangeFn,
  type PaginationState,
  type SortingState,
} from '@tanstack/react-table'
import { ArrowDown, ArrowUp, ArrowUpDown, ChevronLeft, ChevronRight } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
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
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'
import { pageWindow, type SortOrder } from '@/lib/table-state'

export type DataTableLabels = {
  caption: string
  loadingLabel: string
  emptyTitle: string
  emptyDescription?: string
  resultLabel: string
  pageInfo: string
  prevPage: string
  nextPage: string
  pageSizeLabel: string
  itemsPerPage: string
  sortBy: string
}

export type DataTableProps<TData, SortField extends string = string> = {
  columns: ColumnDef<TData, unknown>[]
  data: TData[]
  getRowId?: (row: TData) => string
  /** Allowlist kolom sort yang sama dengan milik backend. */
  sortFields: readonly SortField[]
  sort: SortField
  order: SortOrder
  onSortChange: (field: SortField, order: SortOrder) => void
  page: number
  pageSize: number
  totalItems: number
  totalPages: number
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
  pageSizeOptions?: readonly number[]
  labels: DataTableLabels
  isLoading?: boolean
  isFetching?: boolean
  emptyState?: React.ReactNode
  className?: string
}

const DEFAULT_PAGE_SIZES = [10, 20, 25, 50, 100] as const

/**
 * Tabel data untuk daftar yang dipaginasikan server.
 *
 * Backend memegang pagination, sorting, dan filtering, jadi tabel ini
 * berjalan pada mode manual: baris dan jumlah total datang dari server dan
 * setiap perubahan state diteruskan ke pemanggil.
 */
export function DataTable<TData, SortField extends string = string>({
  columns,
  data,
  getRowId,
  sortFields,
  sort,
  order,
  onSortChange,
  page,
  pageSize,
  totalItems,
  totalPages,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions = DEFAULT_PAGE_SIZES,
  labels,
  isLoading = false,
  isFetching = false,
  emptyState,
  className,
}: DataTableProps<TData, SortField>) {
  const sorting = useMemo<SortingState>(
    () => (sort ? [{ id: sort, desc: order === 'desc' }] : []),
    [sort, order],
  )

  const pagination = useMemo<PaginationState>(
    () => ({ pageIndex: Math.max(page - 1, 0), pageSize }),
    [page, pageSize],
  )

  const handleSortingChange: OnChangeFn<SortingState> = (updater) => {
    const next = typeof updater === 'function' ? updater(sorting) : updater
    const target = next[0]
    if (!target) return
    onSortChange(target.id as SortField, target.desc ? 'desc' : 'asc')
  }

  const handlePaginationChange: OnChangeFn<PaginationState> = (updater) => {
    const next = typeof updater === 'function' ? updater(pagination) : updater
    if (next.pageSize !== pagination.pageSize) {
      onPageSizeChange(next.pageSize)
      return
    }
    onPageChange(next.pageIndex + 1)
  }

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getRowId: getRowId as ((row: TData, index: number) => string) | undefined,
    manualPagination: true,
    manualSorting: true,
    manualFiltering: true,
    rowCount: totalItems,
    state: { pagination, sorting },
    onPaginationChange: handlePaginationChange,
    onSortingChange: handleSortingChange,
  })

  const columnCount = table.getVisibleLeafColumns().length
  const showEmpty = !isLoading && data.length === 0
  const lastPage = Math.max(totalPages, 1)

  return (
    <div
      className={cn(
        'overflow-hidden rounded-2xl border border-[var(--line)] bg-[var(--surface-strong)]',
        className,
      )}
    >
      <div className="relative">
        <Table aria-busy={isFetching || isLoading} aria-label={labels.caption}>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow
                key={headerGroup.id}
                className="border-b border-[var(--line)] bg-[var(--chip-bg)] hover:bg-[var(--chip-bg)]"
              >
                {headerGroup.headers.map((header) => {
                  const sortable = (sortFields as readonly string[]).includes(
                    header.column.id,
                  )
                  const active = sort === header.column.id
                  const ariaSort = active
                    ? order === 'asc'
                      ? 'ascending'
                      : 'descending'
                    : undefined
                  const Icon = active
                    ? order === 'asc'
                      ? ArrowUp
                      : ArrowDown
                    : ArrowUpDown
                  return (
                    <TableHead
                      key={header.id}
                      aria-sort={ariaSort}
                      className="h-11 px-4 text-xs font-semibold tracking-wide text-[var(--sea-ink-soft)] uppercase"
                    >
                      {header.isPlaceholder ? null : sortable ? (
                        <button
                          type="button"
                          className="inline-flex items-center gap-1.5 rounded-md px-1 py-0.5 text-left transition-colors hover:text-[var(--sea-ink)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--ring)]"
                          onClick={() => {
                            const nextOrder: SortOrder =
                              active && order === 'asc' ? 'desc' : 'asc'
                            table.setSorting([
                              { id: header.column.id, desc: nextOrder === 'desc' },
                            ])
                          }}
                        >
                          {flexRender(
                            header.column.columnDef.header,
                            header.getContext(),
                          )}
                          <Icon
                            className={cn(
                              'h-3.5 w-3.5',
                              active
                                ? 'text-[var(--sea-ink)]'
                                : 'text-[var(--sea-ink-soft)] opacity-60',
                            )}
                            aria-hidden="true"
                          />
                          <span className="sr-only">{labels.sortBy}</span>
                        </button>
                      ) : (
                        flexRender(header.column.columnDef.header, header.getContext())
                      )}
                    </TableHead>
                  )
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 5 }, (_, rowIndex) => (
                <TableRow key={`skeleton-${rowIndex}`}>
                  {Array.from({ length: columnCount }, (_, cellIndex) => (
                    <TableCell key={`skeleton-${rowIndex}-${cellIndex}`}>
                      <Skeleton className="h-4 w-full max-w-32" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : showEmpty ? (
              <TableRow>
                <TableCell colSpan={columnCount} className="px-4 py-12">
                  <div className="mx-auto max-w-md text-center">
                    <p className="mb-1 text-sm font-semibold text-[var(--sea-ink)]">
                      {labels.emptyTitle}
                    </p>
                    {labels.emptyDescription ? (
                      <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
                        {labels.emptyDescription}
                      </p>
                    ) : null}
                    {emptyState ? (
                      <div className="mt-4 flex justify-center">{emptyState}</div>
                    ) : null}
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  className={cn(
                    'border-[var(--line)] transition-colors hover:bg-[var(--chip-bg)]',
                    isFetching && 'opacity-70',
                  )}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className="px-4 py-3 text-[var(--sea-ink)]">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <div className="flex flex-col gap-3 border-t border-[var(--line)] px-4 py-3 text-sm text-[var(--sea-ink-soft)] sm:flex-row sm:items-center sm:justify-between">
        <p className="m-0">{labels.resultLabel}</p>
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium uppercase tracking-wide">
              {labels.itemsPerPage}
            </span>
            <Select
              value={String(pageSize)}
              onValueChange={(value) => onPageSizeChange(Number(value))}
            >
              <SelectTrigger className="h-9 w-20" aria-label={labels.pageSizeLabel}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {pageSizeOptions.map((option) => (
                  <SelectItem key={option} value={String(option)}>
                    {option}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[var(--line)] text-[var(--sea-ink)] transition-colors hover:bg-[var(--chip-bg)] disabled:pointer-events-none disabled:opacity-40"
              onClick={() => onPageChange(Math.max(page - 1, 1))}
              disabled={page <= 1 || isFetching}
              aria-label={labels.prevPage}
            >
              <ChevronLeft className="h-4 w-4" aria-hidden="true" />
            </button>
            <nav className="flex items-center gap-1" aria-label={labels.pageInfo}>
              {pageWindow(page, lastPage).map((entry, index) =>
                entry === null ? (
                  <span
                    key={`gap-${index}`}
                    className="px-1 text-xs text-[var(--sea-ink-soft)]"
                    aria-hidden="true"
                  >
                    ...
                  </span>
                ) : (
                  <button
                    key={entry}
                    type="button"
                    onClick={() => onPageChange(entry)}
                    disabled={isFetching}
                    aria-current={entry === page ? 'page' : undefined}
                    aria-label={labels.pageInfo
                      .replace('{page}', String(entry))
                      .replace('{total}', String(lastPage))}
                    className={cn(
                      'inline-flex h-9 min-w-9 items-center justify-center rounded-lg px-2 text-xs font-semibold tabular-nums transition-colors disabled:opacity-40',
                      entry === page
                        ? 'bg-[var(--palm)] text-white'
                        : 'border border-[var(--line)] text-[var(--sea-ink)] hover:bg-[var(--chip-bg)]',
                    )}
                  >
                    {entry}
                  </button>
                ),
              )}
            </nav>
            <button
              type="button"
              className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[var(--line)] text-[var(--sea-ink)] transition-colors hover:bg-[var(--chip-bg)] disabled:pointer-events-none disabled:opacity-40"
              onClick={() => onPageChange(Math.min(page + 1, lastPage))}
              disabled={page >= lastPage || isFetching}
              aria-label={labels.nextPage}
            >
              <ChevronRight className="h-4 w-4" aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
