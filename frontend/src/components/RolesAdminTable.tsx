import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Search } from 'lucide-react'
import { DataTable } from '@/components/DataTable'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useMessages } from '@/i18n/locale-context'
import { EMPLOYEE_SORT_FIELDS } from '@/lib/employees'
import type { EmployeeSortField, EmployeeSortOrder } from '@/lib/employees'
import { formatRoleName } from '@/lib/roles'
import { visibleRange } from '@/lib/table-state'
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
  sort: EmployeeSortField
  order: EmployeeSortOrder
  homebaseId: string | null
  search: string
  onSearchChange: (value: string) => void
  onHomebaseChange: (value: string | null) => void
  onPageSizeChange: (size: RolesPageSize) => void
  onPageChange: (page: number) => void
  onSortChange: (field: EmployeeSortField, order: EmployeeSortOrder) => void
  onSave: (employee: Employee, target: RoleName) => void
  savingId: string | null
  draftRoles: Record<string, RoleName>
  onDraftRoleChange: (employeeId: string, role: RoleName) => void
  unitOptions: Array<{ id: string; name: string | null }>
}

/**
 * Tabel penetapan peran. Daftar pegawai, pencarian, penyaringan unit, dan
 * pengurutan dikerjakan backend; tabel ini menyusun kolom dan meneruskan
 * pilihan pengguna ke mesin tabel bersama.
 */
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
  const [from, to] = visibleRange(page, pageSize, totalItems)
  const numberFormat = useMemo(() => new Intl.NumberFormat(localeTag), [localeTag])

  const currentRole = (employee: Employee): RoleName | '' => {
    const draft = draftRoles[employee.id]
    if (draft) {
      return draft
    }
    const roleName = employee.roleName
    return ROLES.includes(roleName as RoleName) ? (roleName as RoleName) : ''
  }

  const columns = useMemo<ColumnDef<Employee, unknown>[]>(
    () => [
      {
        id: 'nama',
        accessorFn: (row) => row.nama,
        header: t('adminRoles.colName'),
        cell: ({ row }) => (
          <span className="font-medium text-[var(--sea-ink)]">{row.original.nama}</span>
        ),
      },
      {
        id: 'email',
        accessorFn: (row) => row.email,
        header: t('adminRoles.colEmail'),
        cell: ({ row }) => (
          <span className="text-[var(--sea-ink-soft)]">{row.original.email}</span>
        ),
      },
      {
        id: 'homebaseId',
        accessorFn: (row) => row.homebaseId ?? '',
        header: t('adminRoles.colUnit'),
        cell: ({ row }) => (
          <span className="text-[var(--sea-ink-soft)]">
            {row.original.homebaseId
              ? (unitNames.get(row.original.homebaseId) ?? row.original.homebaseId)
              : t('organizations.noParent')}
          </span>
        ),
      },
      {
        id: 'peran',
        header: t('adminRoles.colRole'),
        cell: ({ row }) => {
          const role = currentRole(row.original)
          return (
            <Select
              value={role}
              onValueChange={(value) =>
                onDraftRoleChange(row.original.id, value as RoleName)
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
          )
        },
      },
      {
        id: 'aksi',
        header: () => <span className="sr-only">{t('adminRoles.colAction')}</span>,
        cell: ({ row }) => {
          const role = currentRole(row.original)
          const dirty = role !== '' && role !== row.original.roleName
          return (
            <div className="text-right">
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={!dirty || savingId === row.original.id}
                onClick={() => dirty && onSave(row.original, role as RoleName)}
              >
                {t('adminRoles.save')}
              </Button>
            </div>
          )
        },
      },
    ],
    // currentRole membaca draftRoles, jadi keduanya ikut menjadi dependensi.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, unitNames, draftRoles, savingId, onDraftRoleChange, onSave],
  )

  return (
    <div className="space-y-4">
      <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
        <label className="grid gap-1">
          <span className="text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase">
            {t('adminRoles.searchLabel')}
          </span>
          <div className="relative">
            <Search
              className="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-[var(--sea-ink-soft)]"
              aria-hidden="true"
            />
            <Input
              type="search"
              value={search}
              onChange={(event) => onSearchChange(event.target.value)}
              placeholder={t('adminRoles.searchPlaceholder')}
              className="h-11 pl-9"
            />
          </div>
        </label>
        <label className="grid gap-1">
          <span className="text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase">
            {t('adminRoles.unitLabel')}
          </span>
          <Select
            value={homebaseId ?? 'all'}
            onValueChange={(value) => onHomebaseChange(value === 'all' ? null : value)}
          >
            <SelectTrigger
              className="h-11 min-w-[14rem]"
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
        </label>
      </div>

      <DataTable
        columns={columns}
        data={items}
        getRowId={(row) => row.id}
        sortFields={EMPLOYEE_SORT_FIELDS}
        sort={sort}
        order={order}
        onSortChange={onSortChange}
        page={page}
        pageSize={pageSize}
        totalItems={totalItems}
        totalPages={totalPages}
        onPageChange={onPageChange}
        onPageSizeChange={(size) => onPageSizeChange(size as RolesPageSize)}
        pageSizeOptions={PAGE_SIZES}
        isFetching={isFetching}
        labels={{
          caption: t('adminRoles.tableCaption'),
          loadingLabel: t('audit.refreshing'),
          emptyTitle: t('adminRoles.emptyTitle'),
          emptyDescription: t('adminRoles.emptyDescription'),
          resultLabel: t('common.resultCount', {
            from: numberFormat.format(from),
            to: numberFormat.format(to),
            total: numberFormat.format(totalItems),
          }),
          pageInfo: t('common.pageInfo'),
          prevPage: t('common.prevPage'),
          nextPage: t('common.nextPage'),
          pageSizeLabel: t('common.pageSize'),
          itemsPerPage: t('common.itemsPerPage'),
          sortBy: t('common.sortBy'),
        }}
      />
    </div>
  )
}
