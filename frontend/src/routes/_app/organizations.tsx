import { useEffect, useMemo, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { Plus, ShieldAlert } from 'lucide-react'
import {
  ArchiveOrganizationDialog,
  CreateOrganizationDialog,
  EditOrganizationDialog,
} from '@/components/OrganizationDialogs'
import OrganizationTable, {
  type OrganizationPageSize,
} from '@/components/OrganizationTable'
import LoadingRadar from '@/components/LoadingRadar'
import RetryButton from '@/components/RetryButton'
import { Button } from '@/components/ui/button'
import { useLocale, useMessages } from '@/i18n/locale-context'
import {
  ORGANIZATION_SORT_FIELDS,
  useOrganizations,
  useOrganizationsPage,
  type OrganizationListParams,
  type OrganizationSortField,
} from '@/lib/organizations'
import type { Organization } from '@/validations/schemas'
import { useSession } from '@/lib/session'
import { useTableControls, visibleRange } from '@/lib/table-state'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

export const Route = createFileRoute('/_app/organizations')({
  component: OrganizationPage,
})

const SEARCH_DEBOUNCE_MS = 300

/**
 * Halaman daftar unit organisasi. Backend memegang pagination, sorting, dan
 * filtering, jadi halaman ini hanya menyimpan state kendali tabel.
 */
function OrganizationPage() {
  const t = useMessages()
  const { locale } = useLocale()
  const session = useSession()
  const queryClient = useQueryClient()
  const parentUnits = useOrganizations()

  const permissions = session.data?.permissions ?? []
  const canRead =
    permissions.includes('organization:read') ||
    permissions.includes('organization:read:homebase')
  const canManage = permissions.includes('organization:manage')

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [jenis, setJenis] = useState<string | null>(null)

  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [editingOrganization, setEditingOrganization] = useState<Organization | null>(
    null,
  )
  const [archivingOrganization, setArchivingOrganization] = useState<Organization | null>(
    null,
  )

  const controls = useTableControls<OrganizationSortField>({
    sortFields: ORGANIZATION_SORT_FIELDS,
    initialSort: 'name',
    initialPageSize: 20,
  })
  const { sort, order, page, pageSize, resetPage } = controls

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput.trim())
      resetPage()
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [searchInput, resetPage])

  const params: OrganizationListParams = useMemo(
    () => ({ page, pageSize, search, sort, order, jenis }),
    [page, pageSize, search, sort, order, jenis],
  )

  const query = useOrganizationsPage(params)

  const parentNames = useMemo(() => {
    const map = new Map<string, string>()
    for (const unit of parentUnits.data ?? []) {
      if (unit.name) {
        map.set(unit.id, unit.name)
      }
    }
    return map
  }, [parentUnits.data])

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

  const data = query.data
  const localeTag = locale === 'id' ? 'id-ID' : 'en-US'
  const totalItems = data?.totalItems ?? 0
  const [from, to] = visibleRange(page, pageSize, totalItems)

  const labels = {
    caption: t('organizations.tableCaption'),
    name: t('organizations.colName'),
    type: t('organizations.colType'),
    parent: t('organizations.colParent'),
    status: t('organizations.colStatus'),
    active: t('organizations.statusActive'),
    inactive: t('organizations.statusInactive'),
    noParent: t('organizations.noParent'),
    searchLabel: t('organizations.searchLabel'),
    searchPlaceholder: t('organizations.searchPlaceholder'),
    kindLabel: t('organizations.kindLabel'),
    kindAll: t('organizations.kindAll'),
    emptyTitle: t('organizations.emptyTitle'),
    emptyHint: t('organizations.emptyDescription'),
    loadingLabel: t('common.loadingTable'),
    resultLabel: t('organizations.resultCount', {
      from: new Intl.NumberFormat(localeTag).format(from),
      to: new Intl.NumberFormat(localeTag).format(to),
      total: new Intl.NumberFormat(localeTag).format(totalItems),
    }),
    pageInfo: t('common.pageInfo'),
    prevPage: t('common.prevPage'),
    nextPage: t('common.nextPage'),
    pageSizeLabel: t('common.pageSize'),
    itemsPerPage: t('common.itemsPerPage'),
    sortBy: t('common.sortBy'),
    edit: t('common.edit'),
    archive: t('organizations.archive'),
    actions: t('common.actions'),
    typeUnit: t('organizations.typeUnit'),
    typeFakultas: t('organizations.typeFakultas'),
    typeJurusan: t('organizations.typeJurusan'),
    typeBiro: t('organizations.typeBiro'),
    typeBagian: t('organizations.typeBagian'),
  }

  function handleCreateSuccess(created: Organization) {
    setIsCreateOpen(false)
    queryClient.invalidateQueries({ queryKey: ['organizations'] })
    toast.success(t('organizations.createSuccess', { name: created.name ?? '' }))
  }

  function handleUpdateSuccess(updated: Organization) {
    setEditingOrganization(null)
    queryClient.invalidateQueries({ queryKey: ['organizations'] })
    toast.success(t('organizations.updateSuccess', { name: updated.name ?? '' }))
  }

  function handleArchiveSuccess(name: string) {
    setArchivingOrganization(null)
    queryClient.invalidateQueries({ queryKey: ['organizations'] })
    toast.success(t('organizations.archiveSuccess', { name }))
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
        <div>
          <p className="island-kicker mb-2">{t('organizations.kicker')}</p>
          <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
            {t('organizations.title')}
          </h1>
          <p className="m-0 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('organizations.lead')}
          </p>
        </div>
        {canManage ? (
          <Button
            onClick={() => setIsCreateOpen(true)}
            className="flex items-center gap-2"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            {t('organizations.createTitle')}
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
            {t('organizations.loadingTitle')}
          </p>
        </div>
      ) : query.isError || !data ? (
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('organizations.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('organizations.errorDescription')}
          </p>
          <RetryButton label={t('common.retry')} onClick={() => void query.refetch()} />
        </div>
      ) : data.items.length === 0 ? (
        <div className="island-shell max-w-2xl rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('organizations.emptyTitle')}
          </h2>
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('organizations.emptyDescription')}
          </p>
        </div>
      ) : (
        <OrganizationTable
          organizations={data.items}
          page={page}
          pageSize={pageSize as OrganizationPageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isLoading={query.isPending}
          isFetching={query.isFetching}
          search={searchInput}
          sort={sort}
          order={order}
          kind={jenis}
          onPageChange={controls.setPage}
          onPageSizeChange={(size) => controls.setPageSize(size as OrganizationPageSize)}
          onSearchChange={setSearchInput}
          onSortChange={controls.applySort}
          onKindChange={(value) => {
            setJenis(value)
            resetPage()
          }}
          onEdit={(organization) => setEditingOrganization(organization)}
          onArchive={(organization) => setArchivingOrganization(organization)}
          canManage={canManage}
          labels={labels}
        />
      )}

      <CreateOrganizationDialog
        open={isCreateOpen}
        onClose={() => setIsCreateOpen(false)}
        onSuccess={handleCreateSuccess}
        parentUnits={parentUnits.data ?? []}
      />

      {editingOrganization ? (
        <EditOrganizationDialog
          organization={editingOrganization}
          onClose={() => setEditingOrganization(null)}
          onSuccess={handleUpdateSuccess}
          parentUnits={parentUnits.data ?? []}
          parentNames={parentNames}
        />
      ) : null}

      {archivingOrganization ? (
        <ArchiveOrganizationDialog
          organization={archivingOrganization}
          onClose={() => setArchivingOrganization(null)}
          onSuccess={handleArchiveSuccess}
        />
      ) : null}
    </div>
  )
}
