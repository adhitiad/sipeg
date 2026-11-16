import { useEffect, useMemo, useState } from 'react'
import { ShieldAlert } from 'lucide-react'
import AuditLogTable, { type AuditLogPageSize } from '@/components/AuditLogTable'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import { useLocale, useMessages } from '@/i18n/locale-context'
import {
  AUDIT_LOG_SORT_FIELDS,
  exportAuditLogsCsv,
  useAuditLogs,
  type AuditLogListParams,
  type AuditLogSortField,
} from '@/lib/audit'
import { useSession } from '@/lib/session'
import { useTableControls, visibleRange } from '@/lib/table-state'
import { toast } from 'sonner'

const SEARCH_DEBOUNCE_MS = 300

/**
 * Halaman jejak audit. Penyaringan, pengurutan, dan pemotongan baris
 * dikerjakan backend, jadi halaman ini hanya menyimpan state tabel dan
 * meneruskannya ke kueri.
 */
export default function AuditLogPage() {
  const t = useMessages()
  const { locale } = useLocale()
  const session = useSession()

  const permissions = session.data?.permissions ?? []
  const canRead =
    permissions.includes('audit:read') || permissions.includes('audit:read:homebase')

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [aksiInput, setAksiInput] = useState('')
  const [aksi, setAksi] = useState<string | null>(null)
  const [isExporting, setIsExporting] = useState(false)
  const controls = useTableControls<AuditLogSortField>({
    sortFields: AUDIT_LOG_SORT_FIELDS,
    initialSort: 'createdAt',
    initialOrder: 'desc',
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

  useEffect(() => {
    const timer = setTimeout(() => {
      setAksi(aksiInput.trim() || null)
      resetPage()
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [aksiInput, resetPage])

  const params: AuditLogListParams = useMemo(
    () => ({ page, pageSize, search, aksi, sort, order }),
    [page, pageSize, search, aksi, sort, order],
  )

  const query = useAuditLogs(params)

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
  const formatNumber = (value: number) => new Intl.NumberFormat(localeTag).format(value)

  const labels = {
    caption: t('audit.tableCaption'),
    time: t('audit.colTime'),
    action: t('audit.colAction'),
    actor: t('audit.colActor'),
    target: t('audit.colTarget'),
    note: t('audit.colNote'),
    ip: t('audit.colIp'),
    noValue: t('audit.noValue'),
    emptyTitle: t('audit.emptyTitle'),
    emptyDescription: t('audit.emptyDescription'),
    resultLabel: t('audit.resultCount', {
      from: formatNumber(from),
      to: formatNumber(to),
      total: formatNumber(totalItems),
    }),
    refreshingLabel: t('audit.refreshing'),
    prevPage: t('common.prevPage'),
    nextPage: t('common.nextPage'),
    pageInfo: t('common.pageInfo'),
    searchLabel: t('audit.searchLabel'),
    searchPlaceholder: t('audit.searchPlaceholder'),
    actionLabel: t('audit.actionLabel'),
    actionPlaceholder: t('audit.actionPlaceholder'),
    pageSizeLabel: t('common.pageSize'),
    itemsPerPage: t('common.itemsPerPage'),
    sortBy: t('common.sortBy'),
    exportCsv: t('audit.exportCsv'),
  }

  function handleExport() {
    setIsExporting(true)
    exportAuditLogsCsv({ aksi, search })
      .then(() => {
        toast.success(t('audit.exportSuccess'))
      })
      .catch(() => {
        toast.error(t('audit.exportError'))
      })
      .finally(() => setIsExporting(false))
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('audit.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('audit.title')}
      </h1>
      <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('audit.lead')}
      </p>

      {query.isPending ? (
        <div className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center">
          <LoadingBars size="md" decorative />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('audit.loadingTitle')}
          </p>
        </div>
      ) : query.isError || !data ? (
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('audit.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('audit.errorDescription')}
          </p>
          <RetryButton label={t('common.retry')} onClick={() => void query.refetch()} />
        </div>
      ) : (
        <AuditLogTable
          logs={data.items}
          page={page}
          pageSize={pageSize as AuditLogPageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isFetching={query.isFetching}
          isExporting={isExporting}
          search={searchInput}
          aksi={aksiInput}
          sort={sort}
          order={order}
          onPageChange={controls.setPage}
          onPageSizeChange={(size) => controls.setPageSize(size)}
          onSearchChange={setSearchInput}
          onAksiChange={(value) => setAksiInput(value ?? '')}
          onSortChange={(field, next) => controls.applySort(field, next)}
          onExport={handleExport}
          localeTag={localeTag}
          labels={labels}
        />
      )}
    </div>
  )
}
