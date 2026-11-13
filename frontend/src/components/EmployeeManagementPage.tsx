import { useEffect, useMemo, useState } from 'react'
import { ShieldAlert, Upload } from 'lucide-react'
import EmployeeImportDialog from '@/components/EmployeeImportDialog'
import EmployeeTable from '@/components/EmployeeTable'
import LoadingRadar from '@/components/LoadingRadar'
import RetryButton from '@/components/RetryButton'
import { Button } from '@/components/ui/button'
import { useLocale, useMessages } from '@/i18n/locale-context'
import {
  useEmployees,
  type EmployeeListParams,
  type EmployeeSortField,
  type EmployeeSortOrder,
} from '@/lib/employees'
import { useOrganizations } from '@/lib/organizations'
import { useQueryClient } from '@tanstack/react-query'
import { useSession } from '@/lib/session'

const SEARCH_DEBOUNCE_MS = 300
/**
 * Halaman manajemen pegawai. Daftar yang tampil sudah dipersempit server
 * sesuai peran pemohon, sehingga filter unit di sini hanya mempersempit
 * hasil lebih lanjut, bukan menambah akses.
 */
export default function EmployeeManagementPage() {
  const t = useMessages()
  const { locale } = useLocale()
  const session = useSession()
  const queryClient = useQueryClient()

  const permissions = session.data?.permissions ?? []
  const canReadAny = permissions.includes('employee:read:any')
  const canReadHomebase = permissions.includes('employee:read:homebase')
  const canReadSelf = permissions.includes('employee:read:self')
  const canRead = canReadAny || canReadHomebase || canReadSelf
  const canReadUnits =
    permissions.includes('organization:read') ||
    permissions.includes('organization:read:homebase')
  const canImport =
    permissions.includes('employee:import') ||
    permissions.includes('employee:import:homebase')

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [sort, setSort] = useState<EmployeeSortField>('nama')
  const [order, setOrder] = useState<EmployeeSortOrder>('asc')
  const [homebaseId, setHomebaseId] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState<10 | 20 | 25 | 50 | 100>(20)
  const [isImportOpen, setIsImportOpen] = useState(false)

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
  const units = useOrganizations(canReadUnits && canRead)

  const unitNames = useMemo(() => {
    const map = new Map<string, string>()
    for (const unit of units.data ?? []) {
      if (unit.name) {
        map.set(unit.id, unit.name)
      }
    }
    return map
  }, [units.data])

  if (session.data && !canRead) {
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

  const scopeLabel = canReadAny
    ? t('employees.scopeAny')
    : canReadHomebase
      ? t('employees.scopeHomebase')
      : t('employees.scopeSelf')

  const data = query.data
  const localeTag = locale === 'id' ? 'id-ID' : 'en-US'
  const totalItems = data?.totalItems ?? 0
  const [from, to] =
    totalItems === 0
      ? [0, 0]
      : [(page - 1) * pageSize + 1, Math.min(page * pageSize, totalItems)]

  const labels = {
    caption: t('employees.tableCaption'),
    name: t('employees.colName'),
    nip: t('employees.colNip'),
    unit: t('employees.colUnit'),
    position: t('employees.colPosition'),
    status: t('employees.colStatus'),
    active: t('employees.statusActive'),
    inactive: t('employees.statusInactive'),
    noUnit: t('employees.noUnit'),
    noValue: t('employees.noValue'),
    resultLabel: t('employees.resultCount', {
      from: new Intl.NumberFormat(localeTag).format(from),
      to: new Intl.NumberFormat(localeTag).format(to),
      total: new Intl.NumberFormat(localeTag).format(totalItems),
    }),
    refreshingLabel: t('employees.refreshing'),
    prevPage: t('employees.prevPage'),
    nextPage: t('employees.nextPage'),
    pageInfo: t('employees.pageInfo', {
      page: String(page),
      total: String(Math.max(data?.totalPages ?? 1, 1)),
    }),
    searchLabel: t('employees.searchLabel'),
    searchPlaceholder: t('employees.searchPlaceholder'),
    unitLabel: t('employees.unitLabel'),
    unitAll: t('employees.unitAll'),
    pageSizeLabel: t('employees.pageSize'),
    itemsPerPage: t('employees.itemsPerPage'),
  }

  function handleSortChange(field: EmployeeSortField) {
    if (field === sort) {
      setOrder((current) => (current === 'asc' ? 'desc' : 'asc'))
      return
    }
    setSort(field)
    setOrder('asc')
    setPage(1)
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-3">{t('employees.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('employees.title')}
      </h1>
      <p className="mb-2 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('employees.lead')}
      </p>
      <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <p className="m-0 text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
          {scopeLabel}
        </p>
        {canImport ? (
          <Button
            type="button"
            variant="outline"
            className="gap-2 self-start sm:self-auto"
            onClick={() => setIsImportOpen(true)}
          >
            <Upload className="h-4 w-4" aria-hidden="true" />
            {t('employees.import')}
          </Button>
        ) : null}
      </div>

      {query.isPending ? (
        <div
          role="status"
          aria-live="polite"
          className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center"
        >
          <LoadingRadar size="md" decorative />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('employees.loadingTitle')}
          </p>
        </div>
      ) : query.isError || !data ? (
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('employees.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('employees.errorDescription')}
          </p>
          <RetryButton label={t('common.retry')} onClick={() => void query.refetch()} />
        </div>
      ) : data.items.length === 0 ? (
        <div className="island-shell max-w-2xl rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('employees.emptyTitle')}
          </h2>
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('employees.emptyDescription')}
          </p>
        </div>
      ) : (
        <EmployeeTable
          employees={data.items}
          unitNames={unitNames}
          sort={sort}
          order={order}
          page={page}
          pageSize={pageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isFetching={query.isFetching}
          search={search}
          homebaseId={homebaseId}
          onSortChange={handleSortChange}
          onPageChange={setPage}
          onPageSizeChange={setPageSize}
          onSearchChange={setSearchInput}
          onHomebaseChange={setHomebaseId}
          labels={labels}
        />
      )}
      <EmployeeImportDialog
        open={isImportOpen}
        onClose={() => setIsImportOpen(false)}
        onImported={() => queryClient.invalidateQueries({ queryKey: ['employees'] })}
      />
    </div>
  )
}
