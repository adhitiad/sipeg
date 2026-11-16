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
import { ORGANIZATION_KINDS, ORGANIZATION_SORT_FIELDS } from '@/lib/organizations'
import type { OrganizationSortField, OrganizationSortOrder } from '@/lib/organizations'
import type { Organization } from '@/validations/schemas'

const PAGE_SIZES = [10, 20, 25, 50, 100] as const
export type OrganizationPageSize = (typeof PAGE_SIZES)[number]

type OrganizationTableLabels = DataTableLabels & {
  name: string
  type: string
  parent: string
  status: string
  active: string
  inactive: string
  noParent: string
  searchLabel: string
  searchPlaceholder: string
  kindLabel: string
  kindAll: string
  edit: string
  archive: string
  actions: string
  typeUnit: string
  typeFakultas: string
  typeJurusan: string
  typeBiro: string
  typeBagian: string
  emptyHint: string
}

type OrganizationTableProps = {
  organizations: Organization[]
  page: number
  pageSize: OrganizationPageSize
  totalPages: number
  totalItems: number
  isLoading: boolean
  isFetching: boolean
  search: string
  sort: OrganizationSortField
  order: OrganizationSortOrder
  kind: string | null
  onPageChange: (page: number) => void
  onPageSizeChange: (size: OrganizationPageSize) => void
  onSearchChange: (value: string) => void
  onSortChange: (field: OrganizationSortField, order: OrganizationSortOrder) => void
  onKindChange: (value: string | null) => void
  onEdit: (organization: Organization) => void
  onArchive: (organization: Organization) => void
  canManage: boolean
  labels: OrganizationTableLabels
}

function kindLabel(kind: string, labels: OrganizationTableLabels): string {
  switch (kind) {
    case 'UNIT':
      return labels.typeUnit
    case 'FAKULTAS':
      return labels.typeFakultas
    case 'JURUSAN':
      return labels.typeJurusan
    case 'BIRO':
      return labels.typeBiro
    case 'BAGIAN':
      return labels.typeBagian
    default:
      return kind
  }
}

/**
 * Tabel daftar unit organisasi.
 *
 * Backend memegang pagination, sorting, dan filtering, jadi tabel ini hanya
 * meneruskan pilihan pengguna.
 */
export default function OrganizationTable({
  organizations,
  page,
  pageSize,
  totalPages,
  totalItems,
  isLoading,
  isFetching,
  search,
  sort,
  order,
  kind,
  onPageChange,
  onPageSizeChange,
  onSearchChange,
  onSortChange,
  onKindChange,
  onEdit,
  onArchive,
  canManage,
  labels,
}: OrganizationTableProps) {
  const columns = useMemo<ColumnDef<Organization, unknown>[]>(() => {
    const base: ColumnDef<Organization, unknown>[] = [
      {
        id: 'name',
        accessorFn: (row) => row.name ?? '',
        header: labels.name,
        cell: ({ row }) => (
          <span className="font-medium break-words text-[var(--sea-ink)]">
            {row.original.name ?? row.original.id}
          </span>
        ),
      },
      {
        id: 'jenis',
        accessorFn: (row) => row.jenis,
        header: labels.type,
        cell: ({ row }) => (
          <span className="text-[var(--sea-ink-soft)]">
            {kindLabel(row.original.jenis, labels)}
          </span>
        ),
      },
      {
        id: 'parentId',
        accessorFn: (row) => row.parentId ?? '',
        header: labels.parent,
        cell: ({ row }) => (
          <span className="break-words text-[var(--sea-ink-soft)]">
            {row.original.parentId ?? labels.noParent}
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
            htmlFor="organization-search"
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
              id="organization-search"
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
            {labels.kindLabel}
          </span>
          <Select
            value={kind ?? 'all'}
            onValueChange={(value) => onKindChange(value === 'all' ? null : value)}
          >
            <SelectTrigger className="h-11 min-w-[14rem]" aria-label={labels.kindLabel}>
              <SelectValue placeholder={labels.kindAll} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{labels.kindAll}</SelectItem>
              {ORGANIZATION_KINDS.map((option) => (
                <SelectItem key={option} value={option}>
                  {kindLabel(option, labels)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={organizations}
        getRowId={(row) => row.id}
        sortFields={ORGANIZATION_SORT_FIELDS}
        sort={sort}
        order={order}
        onSortChange={onSortChange}
        page={page}
        pageSize={pageSize}
        totalItems={totalItems}
        totalPages={totalPages}
        onPageChange={onPageChange}
        onPageSizeChange={(size) => onPageSizeChange(size as OrganizationPageSize)}
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
