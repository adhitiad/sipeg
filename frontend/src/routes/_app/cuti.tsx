import { createFileRoute } from '@tanstack/react-router'
import { useEffect, useMemo, useState } from 'react'
import { ShieldAlert, Plus, User } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useMessages } from '@/i18n/locale-context'
import {
  useCutisPage,
  type CutiListParams,
  type CutiSortField,
  type CutiSortOrder,
} from '@/lib/cuti'
import type { Cuti } from '@/validations/schemas'
import { useSession } from '@/lib/session'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import CutiTable from '@/components/CutiTable'
import {
  CreateCutiDialog,
  EditCutiDialog,
  ApproveCutiDialog,
  RejectCutiDialog,
} from '@/components/CutiDialogs'

const SEARCH_DEBOUNCE_MS = 300

export const Route = createFileRoute('/_app/cuti')({
  component: CutiPage,
})

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

  const canRead = canReadAny || canReadHomebase || canReadSelf

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [jenis, setJenis] = useState<string | null>(null)
  const [status, setStatus] = useState<string | null>(null)
  const [sort, setSort] = useState<CutiSortField>('tanggalMulai')
  const [order, setOrder] = useState<CutiSortOrder>('asc')
  const [page, setPage] = useState(1)
  const pageSize = 25
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [editingCuti, setEditingCuti] = useState<Cuti | null>(null)
  const [approvingCuti, setApprovingCuti] = useState<Cuti | null>(null)
  const [rejectingCuti, setRejectingCuti] = useState<Cuti | null>(null)

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput.trim())
      setPage(1)
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [searchInput])

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
  const from = totalItems === 0 ? 0 : (page - 1) * pageSize + 1
  const to = Math.min(page * pageSize, totalItems)

  const labels = {
    caption: t('cuti.tableCaption'),
    user: t('cuti.colUser'),
    jenis: t('cuti.colJenis'),
    tanggalMulai: t('cuti.colTanggalMulai'),
    tanggalSelesai: t('cuti.colTanggalSelesai'),
    status: t('cuti.colStatus'),
    createdAt: t('cuti.colCreatedAt'),
    resultLabel: t('cuti.resultCount', {
      from: String(from),
      to: String(to),
      total: String(totalItems),
    }),
    refreshingLabel: t('audit.refreshing'),
    prevPage: t('audit.prevPage'),
    nextPage: t('audit.nextPage'),
    pageInfo: t('audit.pageInfo', {
      page: String(page),
      total: String(Math.max(data?.totalPages ?? 1, 1)),
    }),
    searchLabel: t('cuti.searchLabel'),
    searchPlaceholder: t('cuti.searchPlaceholder'),
    jenisLabel: t('cuti.jenisLabel'),
    jenisAll: t('jabatan.typeAll'),
    statusLabel: t('cuti.colStatus'),
    statusAll: t('jabatan.typeAll'),
    userIdLabel: t('cuti.userIdLabel'),
    pageSizeLabel: t('organizations.pageSize'),
    itemsPerPage: t('cuti.itemsPerPage', {
      from: String(from),
      to: String(to),
      total: String(totalItems),
    }),
    approve: t('cuti.approveButton'),
    reject: t('cuti.rejectButton'),
    statusMenunggu: t('cuti.statusMenunggu'),
    statusDisetujui: t('cuti.statusDisetujui'),
    statusDitolak: t('cuti.statusDitolak'),
  }

  function handleSortChange(field: CutiSortField) {
    if (field === sort) {
      setOrder((current) => (current === 'asc' ? 'desc' : 'asc'))
      return
    }
    setSort(field)
    setOrder('asc')
    setPage(1)
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
      ) : data.items.length === 0 ? (
        <div className="island-shell max-w-2xl rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('cuti.emptyTitle')}
          </h2>
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('cuti.emptyDescription')}
          </p>
        </div>
      ) : (
        <CutiTable
          cutis={data.items}
          page={page}
          pageSize={pageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          search={searchInput}
          jenis={jenis}
          status={status}
          sort={sort}
          order={order}
          onPageChange={setPage}
          onSearchChange={setSearchInput}
          onSortChange={handleSortChange}
          onJenisChange={setJenis}
          onStatusChange={setStatus}
          onApprove={(cuti) => setApprovingCuti(cuti)}
          onReject={(cuti) => setRejectingCuti(cuti)}
          canApprove={canApprove}
          canReject={canReject}
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
    </div>
  )
}
