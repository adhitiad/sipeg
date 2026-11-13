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
  useJabatansPage,
  type JabatanListParams,
  type JabatanSortField,
  type JabatanSortOrder,
} from '@/lib/jabatan'
import type { Jabatan } from '@/validations/schemas'
import { useSession } from '@/lib/session'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

const SEARCH_DEBOUNCE_MS = 300

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
  const [sort, setSort] = useState<JabatanSortField>('nama')
  const [order, setOrder] = useState<JabatanSortOrder>('asc')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState<JabatanPageSize>(25)
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [editingJabatan, setEditingJabatan] = useState<Jabatan | null>(null)
  const [archivingJabatan, setArchivingJabatan] = useState<Jabatan | null>(null)

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput.trim())
      setPage(1)
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [searchInput])

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
  const from = totalItems === 0 ? 0 : (page - 1) * pageSize + 1
  const to = Math.min(page * pageSize, totalItems)

  const labels = {
    caption: t('jabatan.tableCaption'),
    name: t('jabatan.colName'),
    type: t('jabatan.colType'),
    status: t('organizations.colStatus'),
    active: t('organizations.statusActive'),
    inactive: t('organizations.statusInactive'),
    resultLabel: t('jabatan.resultCount', {
      from: new Intl.NumberFormat(localeTag).format(from),
      to: new Intl.NumberFormat(localeTag).format(to),
      total: new Intl.NumberFormat(localeTag).format(totalItems),
    }),
    refreshingLabel: t('audit.refreshing'),
    prevPage: t('audit.prevPage'),
    nextPage: t('audit.nextPage'),
    pageInfo: t('audit.pageInfo', {
      page: String(page),
      total: String(Math.max(data?.totalPages ?? 1, 1)),
    }),
    searchLabel: t('jabatan.searchLabel'),
    searchPlaceholder: t('jabatan.searchPlaceholder'),
    jenisLabel: t('jabatan.typeLabel'),
    jenisAll: t('jabatan.typeAll'),
    pageSizeLabel: t('organizations.pageSize'),
    itemsPerPage: t('jabatan.itemsPerPage', {
      from: new Intl.NumberFormat(localeTag).format(from),
      to: new Intl.NumberFormat(localeTag).format(to),
      total: new Intl.NumberFormat(localeTag).format(totalItems),
    }),
    edit: t('common.edit'),
    archive: t('organizations.archive'),
    typeStruktural: t('jabatan.typeStruktural'),
    typeFungsional: t('jabatan.typeFungsional'),
  }

  function handleSortChange(field: JabatanSortField) {
    if (field === sort) {
      setOrder((current) => (current === 'asc' ? 'desc' : 'asc'))
      return
    }
    setSort(field)
    setOrder('asc')
    setPage(1)
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
        <div className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center">
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
          pageSize={pageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isFetching={query.isFetching}
          search={searchInput}
          jenis={jenis}
          sort={sort}
          order={order}
          onPageChange={setPage}
          onPageSizeChange={setPageSize}
          onSearchChange={setSearchInput}
          onSortChange={handleSortChange}
          onJenisChange={setJenis}
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
