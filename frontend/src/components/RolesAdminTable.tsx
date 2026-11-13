import { ChevronLeft, ChevronRight } from 'lucide-react'
import SortIcon from '@/components/SortIcon'
import { Button } from '@/components/ui/button'
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
import { useMessages } from '@/i18n/locale-context'
import { formatRoleName } from '@/lib/roles'
import type { Employee } from '@/validations/schemas'

const PAGE_SIZES = [10, 25, 50, 100] as const
export type RolesPageSize = (typeof PAGE_SIZES)[number]

const ROLES = ['admin', 'operator', 'atasan', 'pegawai'] as const
type RoleName = (typeof ROLES)[number]

type RolesAdminTableProps = {
  items: Employee[]
  totalPages: number
  totalItems: number
  page: number
  pageSize: RolesPageSize
  isFetching: boolean
  localeTag: string
  unitNames: Map<string, string>
  sort: 'nama' | 'createdAt' | 'updatedAt'
  order: 'asc' | 'desc'
  homebaseId: string | null
  search: string
  onSearchChange: (value: string) => void
  onHomebaseChange: (value: string | null) => void
  onPageSizeChange: (size: RolesPageSize) => void
  onPageChange: (page: number) => void
  onSortChange: (field: 'nama' | 'createdAt' | 'updatedAt') => void
  onSave: (employee: Employee, target: RoleName) => void
  savingId: string | null
  draftRoles: Record<string, RoleName>
  onDraftRoleChange: (employeeId: string, role: RoleName) => void
  unitOptions: Array<{ id: string; name: string | null }>
}

export default function RolesAdminTable({
  items,
  totalPages,
  totalItems,
  page,
  pageSize,
  isFetching,
  localeTag,
  unitNames,
  sort,
  order,
  homebaseId,
  search,
  onSearchChange,
  onHomebaseChange,
  onPageSizeChange,
  onPageChange,
  onSortChange,
  onSave,
  savingId,
  draftRoles,
  onDraftRoleChange,
  unitOptions,
}: RolesAdminTableProps) {
  const t = useMessages()
  const from = totalItems === 0 ? 0 : (page - 1) * pageSize + 1
  const to = Math.min(page * pageSize, totalItems)

  function currentRole(employee: Employee): RoleName | '' {
    const draft = draftRoles[employee.id]
    if (draft) {
      return draft
    }
    const roleName = employee.roleName
    return ROLES.includes(roleName as RoleName) ? (roleName as RoleName) : ''
  }

  return (
    <div className="island-shell overflow-hidden rounded-2xl">
      <div className="border-b border-[var(--line)] px-4 py-3 sm:flex sm:items-center sm:justify-between gap-3">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:gap-4">
          <label
            htmlFor="admin-roles-search"
            className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]"
          >
            {t('adminRoles.searchLabel')}
          </label>
          <Input
            id="admin-roles-search"
            type="search"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder={t('adminRoles.searchPlaceholder')}
            className="h-11 w-full sm:w-64"
          />
        </div>
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center">
          <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
            {t('adminRoles.unitLabel')}
          </span>
          <Select
            value={homebaseId ?? 'all'}
            onValueChange={(value) => onHomebaseChange(value === 'all' ? null : value)}
          >
            <SelectTrigger
              className="h-11 min-w-[12rem]"
              aria-label={t('adminRoles.unitLabel')}
            >
              <SelectValue placeholder={t('adminRoles.unitAll')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('adminRoles.unitAll')}</SelectItem>
              {unitOptions.map((unit) => (
                <SelectItem key={unit.id} value={unit.id}>
                  {unit.name ?? unit.id}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center">
          <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
            {t('organizations.pageSize')}
          </span>
          <Select
            value={String(pageSize)}
            onValueChange={(value) => onPageSizeChange(Number(value) as RolesPageSize)}
          >
            <SelectTrigger className="h-11 w-20" aria-label={t('organizations.pageSize')}>
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

      <Table className="min-w-[52rem]">
        <TableCaption className="sr-only">{t('adminRoles.tableCaption')}</TableCaption>
        <TableHeader>
          <TableRow className="border-[var(--line)] hover:bg-transparent">
            <TableHead scope="col" className="px-4 py-3">
              <button
                type="button"
                onClick={() => onSortChange('nama')}
                className="island-kicker inline-flex items-center gap-1"
              >
                {t('adminRoles.colName')}
                <SortIcon active={sort === 'nama'} order={order} />
              </button>
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {t('adminRoles.colEmail')}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {t('adminRoles.colUnit')}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3">
              {t('adminRoles.colRole')}
            </TableHead>
            <TableHead scope="col" className="px-4 py-3 text-right">
              <span className="sr-only">{t('adminRoles.colAction')}</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody aria-busy={isFetching}>
          {items.map((employee) => {
            const role = currentRole(employee)
            const dirty = role !== '' && role !== employee.roleName
            return (
              <TableRow key={employee.id} className="border-[var(--line)] last:border-0">
                <TableCell className="px-4 py-3 font-medium text-[var(--sea-ink)]">
                  {employee.nama}
                </TableCell>
                <TableCell className="px-4 py-3 text-[var(--sea-ink-soft)]">
                  {employee.email}
                </TableCell>
                <TableCell className="px-4 py-3 text-[var(--sea-ink-soft)]">
                  {employee.homebaseId
                    ? (unitNames.get(employee.homebaseId) ?? employee.homebaseId)
                    : t('organizations.noParent')}
                </TableCell>
                <TableCell className="px-4 py-3">
                  <Select
                    value={role}
                    onValueChange={(value) =>
                      onDraftRoleChange(employee.id, value as RoleName)
                    }
                  >
                    <SelectTrigger
                      className="h-10 w-36"
                      aria-label={t('adminRoles.selectRole')}
                    >
                      <SelectValue placeholder={t('adminRoles.selectRole')} />
                    </SelectTrigger>
                    <SelectContent>
                      {ROLES.map((roleName) => (
                        <SelectItem key={roleName} value={roleName}>
                          {formatRoleName(roleName, t)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </TableCell>
                <TableCell className="px-4 py-3 text-right">
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    disabled={!dirty || savingId === employee.id}
                    onClick={() => dirty && onSave(employee, role as RoleName)}
                  >
                    {t('adminRoles.save')}
                  </Button>
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>

      <div className="flex flex-col gap-3 border-t border-[var(--line)] px-4 py-3 text-sm sm:flex-row sm:items-center sm:justify-between">
        <p className="m-0 text-[var(--sea-ink-soft)]">
          {t('adminRoles.itemsPerPage', {
            from: new Intl.NumberFormat(localeTag).format(from),
            to: new Intl.NumberFormat(localeTag).format(to),
            total: new Intl.NumberFormat(localeTag).format(totalItems),
          })}
          {isFetching ? ` · ${t('audit.refreshing')}` : ''}
        </p>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
            aria-label={t('audit.prevPage')}
            className="h-11 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-4 text-xs font-semibold text-[var(--sea-ink)] disabled:cursor-not-allowed disabled:opacity-50 sm:h-9"
          >
            <ChevronLeft className="h-4 w-4" aria-hidden="true" />
          </button>
          {[...Array(totalPages)].map((_, index) => {
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
            aria-label={t('audit.nextPage')}
            className="h-11 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-4 text-xs font-semibold text-[var(--sea-ink)] disabled:cursor-not-allowed disabled:opacity-50 sm:h-9"
          >
            <ChevronRight className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <span className="text-xs text-[var(--sea-ink-soft)]">
          {t('audit.pageInfo', {
            page: String(page),
            total: String(totalPages),
          })}
        </span>
      </div>
    </div>
  )
}
