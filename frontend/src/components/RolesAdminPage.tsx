import { useEffect, useMemo, useState } from 'react'
import { ShieldAlert } from 'lucide-react'
import LoadingRadar from '@/components/LoadingRadar'
import RetryButton from '@/components/RetryButton'
import RolesAdminTable, { type RolesPageSize } from '@/components/RolesAdminTable'
import { useLocale, useMessages } from '@/i18n/locale-context'
import {
  assignEmployeeRole,
  useEmployees,
  type EmployeeListParams,
  type EmployeeSortField,
  type EmployeeSortOrder,
} from '@/lib/employees'
import { useOrganizations } from '@/lib/organizations'
import { formatRoleName } from '@/lib/roles'
import type { Employee } from '@/validations/schemas'
import { useSession } from '@/lib/session'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

const SEARCH_DEBOUNCE_MS = 300
type RoleName = 'admin' | 'operator' | 'atasan' | 'pegawai'

export default function RolesAdminPage() {
  const t = useMessages()
  const { locale } = useLocale()
  const session = useSession()
  const queryClient = useQueryClient()

  const permissions = session.data?.permissions ?? []
  const canManage = permissions.includes('employee:role:manage')

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [sort, setSort] = useState<EmployeeSortField>('nama')
  const [order, setOrder] = useState<EmployeeSortOrder>('asc')
  const [homebaseId, setHomebaseId] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState<RolesPageSize>(25)
  const [draftRoles, setDraftRoles] = useState<Record<string, RoleName>>({})
  const [savingId, setSavingId] = useState<string | null>(null)

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput.trim())
      setPage(1)
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [searchInput])

  const params: EmployeeListParams = useMemo(
    () => ({ page, pageSize, search, sort, order, homebaseId }),
    [page, pageSize, search, sort, order, homebaseId],
  )

  const query = useEmployees(params)
  const units = useOrganizations(true)

  const unitNames = useMemo(() => {
    const map = new Map<string, string>()
    for (const unit of units.data ?? []) {
      if (unit.name) {
        map.set(unit.id, unit.name)
      }
    }
    return map
  }, [units.data])

  const unitOptions = useMemo(
    () => (units.data ?? []).map((unit) => ({ id: unit.id, name: unit.name })),
    [units.data],
  )

  if (session.data && !canManage) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <ShieldAlert
            className="mx-auto mb-3 h-8 w-8 text-[var(--sea-ink-soft)]"
            aria-hidden="true"
          />
          <h1 className="display-title mb-2 text-2xl font-bold text-[var(--sea-ink)]">
            {t('state.noAccess.title')}
          </h1>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('state.noAccess.description')}
          </p>
        </section>
      </div>
    )
  }

  const data = query.data
  const localeTag = locale === 'id' ? 'id-ID' : 'en-US'
  const totalItems = data?.totalItems ?? 0
  const totalPages = Math.max(data?.totalPages ?? 1, 1)

  function handleSortChange(field: EmployeeSortField) {
    if (field === sort) {
      setOrder((current) => (current === 'asc' ? 'desc' : 'asc'))
      return
    }
    setSort(field)
    setOrder('asc')
    setPage(1)
  }

  function handleSave(employee: Employee, target: RoleName) {
    setSavingId(employee.id)
    assignEmployeeRole(employee.id, target)
      .then(() => {
        toast.success(
          t('adminRoles.saveSuccess', {
            name: employee.nama,
            role: formatRoleName(target, t),
          }),
        )
        queryClient.invalidateQueries({ queryKey: ['employees'] })
      })
      .catch(() => {
        toast.error(t('adminRoles.error'))
      })
      .finally(() => setSavingId(null))
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('adminRoles.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('adminRoles.title')}
      </h1>
      <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('adminRoles.lead')}
      </p>

      {query.isPending ? (
        <div className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center">
          <LoadingRadar size="md" decorative />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('adminRoles.loadingTitle')}
          </p>
        </div>
      ) : query.isError || !data ? (
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('adminRoles.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('adminRoles.errorDescription')}
          </p>
          <RetryButton label={t('common.retry')} onClick={() => void query.refetch()} />
        </div>
      ) : data.items.length === 0 ? (
        <div className="island-shell max-w-2xl rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('adminRoles.emptyTitle')}
          </h2>
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('adminRoles.emptyDescription')}
          </p>
        </div>
      ) : (
        <RolesAdminTable
          items={data.items}
          totalPages={totalPages}
          totalItems={totalItems}
          page={page}
          pageSize={pageSize}
          isFetching={query.isFetching}
          localeTag={localeTag}
          unitNames={unitNames}
          sort={sort}
          order={order}
          homebaseId={homebaseId}
          search={searchInput}
          onSearchChange={setSearchInput}
          onHomebaseChange={(value) => {
            setHomebaseId(value)
            setPage(1)
          }}
          onPageSizeChange={setPageSize}
          onPageChange={setPage}
          onSortChange={handleSortChange}
          onSave={handleSave}
          savingId={savingId}
          draftRoles={draftRoles}
          onDraftRoleChange={(employeeId, role) =>
            setDraftRoles((current) => ({ ...current, [employeeId]: role }))
          }
          unitOptions={unitOptions}
        />
      )}
    </div>
  )
}
