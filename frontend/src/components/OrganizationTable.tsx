import { ChevronLeft, ChevronRight, Pencil, Archive } from 'lucide-react'
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
import { ORGANIZATION_KINDS } from '@/lib/organizations'
import type { Organization } from '@/validations/schemas'

const PAGE_SIZES = [10, 20, 25, 50, 100] as const
export type OrganizationPageSize = (typeof PAGE_SIZES)[number]

type OrganizationTableProps = {
  organizations: Organization[]
  page: number
  pageSize: OrganizationPageSize
  totalPages: number
  totalItems: number
  isFetching: boolean
  search: string
  sort: 'name' | 'jenis' | 'createdAt' | 'updatedAt'
  order: 'asc' | 'desc'
  kind: string | null
  onPageChange: (page: number) => void
  onPageSizeChange: (size: OrganizationPageSize) => void
  onSearchChange: (value: string) => void
  onSortChange: (field: 'name' | 'jenis' | 'createdAt' | 'updatedAt') => void
  onKindChange: (value: string | null) => void
  onEdit: (organization: Organization) => void
  onArchive: (organization: Organization) => void
  canManage: boolean
  labels: {
    caption: string
    name: string
    type: string
    parent: string
    status: string
    active: string
    inactive: string
    noParent: string
    resultLabel: string
    refreshingLabel: string
    prevPage: string
    nextPage: string
    pageInfo: string
    searchLabel: string
    searchPlaceholder: string
    kindLabel: string
    kindAll: string
    pageSizeLabel: string
    itemsPerPage: string
    edit: string
    archive: string
    typeUnit: string
    typeFakultas: string
    typeJurusan: string
    typeBiro: string
    typeBagian: string
  }
}

function kindLabel(kind: string, labels: OrganizationTableProps['labels']): string {
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

function pageRange(totalItems: number, page: number, pageSize: number): [number, number] {
  if (totalItems === 0) {
    return [0, 0]
  }
  const from = (page - 1) * pageSize + 1
  const to = Math.min(page * pageSize, totalItems)
  return [from, to]
}

export default function OrganizationTable({
  organizations,
  page,
  pageSize,
  totalPages,
  totalItems,
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
  const [from, to] = pageRange(totalItems, page, pageSize)

  return (
    <div className="island-shell overflow-hidden rounded-2xl">
      <div className="border-b border-[var(--line)] px-4 py-3 sm:flex sm:items-center sm:justify-between gap-3">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center">
          <label
            htmlFor="organization-search"
            className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
          >
            {labels.searchLabel}
          </label>
          <Input
            id="organization-search"
            type="search"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder={labels.searchPlaceholder}
            className="h-11 w-full sm:w-64"
          />
        </div>

        <div className="flex flex-col gap-1 sm:flex-row sm:items-center">
          <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
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

        <div className="flex flex-col gap-1 sm:flex-row sm:items-center">
          <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
            {labels.pageSizeLabel}
          </span>
          <Select
            value={String(pageSize)}
            onValueChange={(value) =>
              onPageSizeChange(value as unknown as OrganizationPageSize)
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

      <Table className="min-w-[40rem]">
        <TableCaption className="sr-only">{labels.caption}</TableCaption>
        <TableHeader>
          <TableRow className="border-[var(--line)] hover:bg-transparent">
            <TableHead scope="col" className="px-4 py-3">
              <button
                type="button"
                onClick={() => onSortChange('name')}
                aria-label={labels.name}
                className="island-kicker inline-flex items-center gap-1"
              >
                {labels.name}
                <SortIcon active={sort === 'name'} order={order} />
              </button>
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              <button
                type="button"
                onClick={() => onSortChange('jenis')}
                aria-label={labels.type}
                className="island-kicker inline-flex items-center gap-1"
              >
                {labels.type}
                <SortIcon active={sort === 'jenis'} order={order} />
              </button>
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.parent}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {labels.status}
            </TableHead>
            {canManage ? (
              <TableHead scope="col" className="px-4 py-3 text-right">
                <span className="sr-only">Aksi</span>
              </TableHead>
            ) : null}
          </TableRow>
        </TableHeader>
        <TableBody aria-busy={isFetching}>
          {organizations.map((organization) => (
            <TableRow
              key={organization.id}
              className="border-[var(--line)] last:border-0 hover:bg-transparent"
            >
              <TableCell className="px-4 py-3 font-medium text-[var(--sea-ink)] break-words">
                {organization.name ?? organization.id}
              </TableCell>
              <TableCell className="px-4 py-3 text-[var(--sea-ink-soft)]">
                {kindLabel(organization.jenis, labels)}
              </TableCell>
              <TableCell className="px-4 py-3 text-[var(--sea-ink-soft)] break-words">
                {organization.parentId ?? labels.noParent}
              </TableCell>
              <TableCell className="px-4 py-3">
                <span
                  className={`font-medium text-xs ${
                    organization.aktif ? 'text-[var(--sea-ink)]' : 'text-destructive'
                  }`}
                >
                  {organization.aktif ? labels.active : labels.inactive}
                </span>
              </TableCell>
              {canManage ? (
                <TableCell className="px-4 py-3 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <button
                      type="button"
                      onClick={() => onEdit(organization)}
                      className="inline-flex items-center gap-1.5 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-1 text-xs font-semibold text-[var(--sea-ink)] transition hover:bg-[var(--link-bg-hover)]"
                    >
                      <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
                      {labels.edit}
                    </button>
                    <button
                      type="button"
                      onClick={() => onArchive(organization)}
                      className="inline-flex items-center gap-1.5 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-1 text-xs font-semibold text-destructive transition hover:bg-[var(--link-bg-hover)]"
                    >
                      <Archive className="h-3.5 w-3.5" aria-hidden="true" />
                      {labels.archive}
                    </button>
                  </div>
                </TableCell>
              ) : null}
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
                aria-label={n === page ? 'Halaman ' + String(n) : undefined}
                aria-current={active ? 'page' : undefined}
                className={
                  active
                    ? 'h-10 w-10 rounded-full bg-[var(--lagoon)] text-white shadow-[0_0_0_1px_var(--lagoon)_inset] text-xs font-semibold'
                    : 'h-10 w-10 rounded-full border border-[var(--line)] bg-[var(--chip-bg)] text-[var(--sea-ink)] text-xs font-semibold hover:bg-[var(--surface-strong)] disabled:opacity-50'
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
