import { useEffect, useMemo, useState } from 'react'
import { ShieldAlert, Plus } from 'lucide-react'
import JabatanTable, { type JabatanPageSize } from '@/components/JabatanTable'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import {
  ArchiveJabatanDialog,
  CreateJabatanDialog,
  EditJabatanDialog,
} from '@/components/JabatanDialogs'
import { Button } from '@/components/ui/button'
import { useLocale, useMessages } from '@/i18n/locale-context'
import {
  JABATAN_SORT_FIELDS,
  useJabatansPage,
  type JabatanListParams,
  type JabatanSortField,
} from '@/lib/jabatan'
import type { Jabatan } from '@/validations/schemas'
import { useSession } from '@/lib/session'
import { useTableControls, visibleRange } from '@/lib/table-state'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

const SEARCH_DEBOUNCE_MS = 300

/**
 * Halaman master jabatan. Backend memegang pagination, sorting, dan filtering,
 * jadi halaman ini hanya menyimpan state kendali tabel.
 */
export default function JabatanPage() {
  const t = useMessages()
  const { locale } = useLocale()
  const session = useSession()
  const queryClient = useQueryClient()

  const permissions = session.data?.permissions ?? []
  const canRead = permissions.includes('jabatan:read')
  const canManage = permissions.includes('jabatan:manage')

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [jenis, setJenis] = useState<string | null>(null)
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [editingJabatan, setEditingJabatan] = useState<Jabatan | null>(null)
  const [archivingJabatan, setArchivingJabatan] = useState<Jabatan | null>(null)
  const controls = useTableControls<JabatanSortField>({
    sortFields: JABATAN_SORT_FIELDS,
    initialSort: 'nama',
    initialPageSize: 25,
  })
  const { sort, order, page, pageSize, resetPage } = controls

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput.trim())
      resetPage()
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [searchInput, resetPage])

  const params: JabatanListParams = useMemo(
    () => ({ page, pageSize, search, jenis, sort, order }),
    [page, pageSize, search, jenis, sort, order],
  )

  const query = useJabatansPage(params)

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
    caption: t('jabatan.tableCaption'),
    name: t('jabatan.colName'),
    type: t('jabatan.colType'),
    status: t('organizations.colStatus'),
    active: t('organizations.statusActive'),
    inactive: t('organizations.statusInactive'),
    searchLabel: t('jabatan.searchLabel'),
    searchPlaceholder: t('jabatan.searchPlaceholder'),
    jenisLabel: t('jabatan.typeLabel'),
    jenisAll: t('jabatan.typeAll'),
    emptyTitle: t('jabatan.emptyTitle'),
    emptyHint: t('jabatan.emptyDescription'),
    loadingLabel: t('common.loadingTable'),
    resultLabel: t('jabatan.resultCount', {
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
    typeStruktural: t('jabatan.typeStruktural'),
    typeFungsional: t('jabatan.typeFungsional'),
  }

  function handleCreateSuccess(created: Jabatan) {
    setIsCreateOpen(false)
    queryClient.invalidateQueries({ queryKey: ['jabatans'] })
    toast.success(t('jabatan.createSuccess', { name: created.nama }))
  }

  function handleUpdateSuccess(updated: Jabatan) {
    setEditingJabatan(null)
    queryClient.invalidateQueries({ queryKey: ['jabatans'] })
    toast.success(t('jabatan.updateSuccess', { name: updated.nama }))
  }

  function handleArchiveSuccess(name: string) {
    setArchivingJabatan(null)
    queryClient.invalidateQueries({ queryKey: ['jabatans'] })
    toast.success(t('jabatan.archiveSuccess', { name }))
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('jabatan.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('jabatan.title')}
      </h1>
      <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <p className="m-0 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
          {t('jabatan.lead')}
        </p>
        {canManage ? (
          <Button
            type="button"
            className="gap-2 self-start sm:self-auto"
            onClick={() => setIsCreateOpen(true)}
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            {t('jabatan.createTitle')}
          </Button>
        ) : null}
      </div>

      {query.isPending ? (
        <div
          role="status"
          aria-live="polite"
          className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center"
        >
          <LoadingBars size="md" decorative />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('jabatan.loadingTitle')}
          </p>
        </div>
      ) : query.isError || !data ? (
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('jabatan.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('jabatan.errorDescription')}
          </p>
          <RetryButton label={t('common.retry')} onClick={() => void query.refetch()} />
        </div>
      ) : data.items.length === 0 ? (
        <div className="island-shell max-w-2xl rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('jabatan.emptyTitle')}
          </h2>
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('jabatan.emptyDescription')}
          </p>
        </div>
      ) : (
        <JabatanTable
          jabatans={data.items}
          page={page}
          pageSize={pageSize as JabatanPageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isLoading={query.isPending}
          isFetching={query.isFetching}
          search={searchInput}
          jenis={jenis}
          sort={sort}
          order={order}
          onPageChange={controls.setPage}
          onPageSizeChange={(size) => controls.setPageSize(size as JabatanPageSize)}
          onSearchChange={setSearchInput}
          onSortChange={controls.applySort}
          onJenisChange={(value) => {
            setJenis(value)
            resetPage()
          }}
          onEdit={(jabatan) => setEditingJabatan(jabatan)}
          onArchive={(jabatan) => setArchivingJabatan(jabatan)}
          canManage={canManage}
          labels={labels}
        />
      )}

      <CreateJabatanDialog
        open={isCreateOpen}
        onClose={() => setIsCreateOpen(false)}
        onSuccess={handleCreateSuccess}
      />

      {editingJabatan ? (
        <EditJabatanDialog
          jabatan={editingJabatan}
          onClose={() => setEditingJabatan(null)}
          onSuccess={handleUpdateSuccess}
        />
      ) : null}

      {archivingJabatan ? (
        <ArchiveJabatanDialog
          jabatan={archivingJabatan}
          onClose={() => setArchivingJabatan(null)}
          onSuccess={handleArchiveSuccess}
        />
      ) : null}
    </div>
  )
}
