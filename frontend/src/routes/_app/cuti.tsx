import { createFileRoute } from '@tanstack/react-router'
import { useEffect, useMemo, useState } from 'react'
import { ShieldAlert, Plus, User } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useMessages } from '@/i18n/locale-context'
import {
  CUTI_SORT_FIELDS,
  useCutisPage,
  type CutiListParams,
  type CutiSortField,
} from '@/lib/cuti'
import type { Cuti } from '@/validations/schemas'
import { useSession } from '@/lib/session'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import CutiTable, { type CutiPageSize } from '@/components/CutiTable'
import {
  CreateCutiDialog,
  EditCutiDialog,
  ApproveCutiDialog,
  RejectCutiDialog,
  CancelCutiDialog,
} from '@/components/CutiDialogs'
import { useTableControls, visibleRange } from '@/lib/table-state'

const SEARCH_DEBOUNCE_MS = 300

export const Route = createFileRoute('/_app/cuti')({
  component: CutiPage,
})

/**
 * Halaman pengajuan cuti. Daftar yang tampil sudah dipersempit server sesuai
 * peran pemohon; penyaringan dan pengurutan di sini hanya mempersempit hasil
 * lebih lanjut, bukan menambah akses.
 */
function CutiPage() {
  const t = useMessages()
  const session = useSession()
  const queryClient = useQueryClient()

  const permissions = session.data?.permissions ?? []
  const canReadAny = permissions.includes('cuti:read:any')
  const canReadHomebase = permissions.includes('cuti:read:homebase')
  const canReadSelf = permissions.includes('cuti:read:self')
  const canCreate =
    permissions.includes('cuti:create') || permissions.includes('cuti:create:homebase')
  const canApprove =
    permissions.includes('cuti:approve:any') ||
    permissions.includes('cuti:approve:homebase')
  const canReject = canApprove
  const canCancelSelf = permissions.includes('cuti:cancel:self')

  const canRead = canReadAny || canReadHomebase || canReadSelf

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [jenis, setJenis] = useState<string | null>(null)
  const [status, setStatus] = useState<string | null>(null)
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [editingCuti, setEditingCuti] = useState<Cuti | null>(null)
  const [approvingCuti, setApprovingCuti] = useState<Cuti | null>(null)
  const [rejectingCuti, setRejectingCuti] = useState<Cuti | null>(null)
  const [cancelingCuti, setCancelingCuti] = useState<Cuti | null>(null)
  const controls = useTableControls<CutiSortField>({
    sortFields: CUTI_SORT_FIELDS,
    initialSort: 'tanggalMulai',
    initialOrder: 'asc',
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

  const params: CutiListParams = useMemo(
    () => ({ page, pageSize, search, jenis, status, sort, order }),
    [page, pageSize, search, jenis, status, sort, order],
  )

  const query = useCutisPage(params)

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
  const totalItems = data?.totalItems ?? 0
  const [from, to] = visibleRange(page, pageSize, totalItems)

  const labels = {
    caption: t('cuti.tableCaption'),
    user: t('cuti.colUser'),
    jenis: t('cuti.colJenis'),
    tanggalMulai: t('cuti.colTanggalMulai'),
    tanggalSelesai: t('cuti.colTanggalSelesai'),
    status: t('cuti.colStatus'),
    createdAt: t('cuti.colCreatedAt'),
    action: t('cuti.colAction'),
    emptyTitle: t('cuti.emptyTitle'),
    emptyDescription: t('cuti.emptyDescription'),
    resultLabel: t('cuti.resultCount', {
      from: String(from),
      to: String(to),
      total: String(totalItems),
    }),
    refreshingLabel: t('audit.refreshing'),
    prevPage: t('common.prevPage'),
    nextPage: t('common.nextPage'),
    pageInfo: t('common.pageInfo'),
    searchLabel: t('cuti.searchLabel'),
    searchPlaceholder: t('cuti.searchPlaceholder'),
    jenisLabel: t('cuti.jenisLabel'),
    jenisAll: t('cuti.jenisAll'),
    statusLabel: t('cuti.colStatus'),
    statusAll: t('cuti.statusAll'),
    pageSizeLabel: t('common.pageSize'),
    itemsPerPage: t('common.itemsPerPage'),
    sortBy: t('common.sortBy'),
    approve: t('cuti.approveButton'),
    reject: t('cuti.rejectButton'),
    cancel: t('cuti.cancelButton'),
    statusMenunggu: t('cuti.statusMenunggu'),
    statusDisetujui: t('cuti.statusDisetujui'),
    statusDitolak: t('cuti.statusDitolak'),
    statusBatal: t('cuti.statusBatal'),
    jenisLabels: {
      TAHUNAN: t('cuti.jenisTahunan'),
      PANJANG: t('cuti.jenisPanjang'),
      BERSALIN: t('cuti.jenisBersalin'),
      SAKIT: t('cuti.jenisSakit'),
      SAKIT_BERKEPANJANGAN: t('cuti.jenisSakitBerkepanjangan'),
      IZIN_PENTING: t('cuti.jenisIzinPenting'),
      POLITIK: t('cuti.jenisPolitik'),
    },
  }

  function handleCreateSuccess() {
    setIsCreateOpen(false)
    queryClient.invalidateQueries({ queryKey: ['cutis'] })
    toast.success(t('cuti.createSuccess'))
  }

  function handleUpdateSuccess() {
    setEditingCuti(null)
    queryClient.invalidateQueries({ queryKey: ['cutis'] })
    toast.success(t('cuti.updateSuccess'))
  }

  function handleApproveSuccess() {
    setApprovingCuti(null)
    queryClient.invalidateQueries({ queryKey: ['cutis'] })
    toast.success(t('cuti.approveSuccess'))
  }

  function handleRejectSuccess() {
    setRejectingCuti(null)
    queryClient.invalidateQueries({ queryKey: ['cutis'] })
    toast.success(t('cuti.rejectSuccess'))
  }

  function handleCancelSuccess() {
    setCancelingCuti(null)
    queryClient.invalidateQueries({ queryKey: ['cutis'] })
    toast.success(t('cuti.cancelSuccess'))
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('cuti.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('cuti.title')}
      </h1>
      <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <p className="m-0 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
          {t('cuti.lead')}
        </p>
        {canCreate ? (
          <Button
            type="button"
            className="gap-2 self-start sm:self-auto"
            onClick={() => setIsCreateOpen(true)}
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            {t('cuti.createTitle')}
          </Button>
        ) : null}
      </div>

      {query.isPending ? (
        <div className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center">
          <User className="h-8 w-8 text-[var(--sea-ink-soft)]" aria-hidden="true" />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('cuti.loadingTitle')}
          </p>
        </div>
      ) : query.isError || !data ? (
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('cuti.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('cuti.errorDescription')}
          </p>
          <Button type="button" variant="outline" onClick={() => void query.refetch()}>
            {t('common.retry')}
          </Button>
        </div>
      ) : (
        <CutiTable
          cutis={data.items}
          page={page}
          pageSize={pageSize as CutiPageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isFetching={query.isFetching}
          search={searchInput}
          jenis={jenis}
          status={status}
          sort={sort}
          order={order}
          onPageChange={controls.setPage}
          onPageSizeChange={(size) => controls.setPageSize(size)}
          onSearchChange={setSearchInput}
          onSortChange={(field, next) => controls.applySort(field, next)}
          onJenisChange={(value) => {
            setJenis(value)
            resetPage()
          }}
          onStatusChange={(value) => {
            setStatus(value)
            resetPage()
          }}
          onApprove={(cuti) => setApprovingCuti(cuti)}
          onReject={(cuti) => setRejectingCuti(cuti)}
          onCancel={(cuti) => setCancelingCuti(cuti)}
          canApprove={canApprove}
          canReject={canReject}
          canCancelSelf={canCancelSelf}
          labels={labels}
        />
      )}

      <CreateCutiDialog
        open={isCreateOpen}
        onClose={() => setIsCreateOpen(false)}
        onSuccess={handleCreateSuccess}
      />

      {editingCuti ? (
        <EditCutiDialog
          cuti={editingCuti}
          onClose={() => setEditingCuti(null)}
          onSuccess={handleUpdateSuccess}
        />
      ) : null}

      {approvingCuti ? (
        <ApproveCutiDialog
          cuti={approvingCuti}
          onClose={() => setApprovingCuti(null)}
          onSuccess={handleApproveSuccess}
        />
      ) : null}

      {rejectingCuti ? (
        <RejectCutiDialog
          cuti={rejectingCuti}
          onClose={() => setRejectingCuti(null)}
          onSuccess={handleRejectSuccess}
        />
      ) : null}

      {cancelingCuti ? (
        <CancelCutiDialog
          cuti={cancelingCuti}
          onClose={() => setCancelingCuti(null)}
          onSuccess={handleCancelSuccess}
        />
      ) : null}
    </div>
  )
}
