import { useMemo } from 'react'
import { ShieldAlert } from 'lucide-react'
import ArsipTable, { type ArsipPageSize } from '@/components/ArsipTable'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import { useMessages } from '@/i18n/locale-context'
import {
  ARSIP_SORT_FIELDS,
  useArsip,
  type ArsipListParams,
  type ArsipSortField,
} from '@/lib/arsip'
import { useSession } from '@/lib/session'
import { useTableControls, visibleRange } from '@/lib/table-state'

export default function ArsipPage() {
  const t = useMessages()
  const session = useSession()

  const permissions = session.data?.permissions ?? []
  const canRead = permissions.some((p) => p.startsWith('arsip:read'))

  const controls = useTableControls<ArsipSortField>({
    sortFields: ARSIP_SORT_FIELDS,
    initialSort: 'createdAt',
    initialPageSize: 25,
  })
  const { sort, order, page, pageSize } = controls

  const params: ArsipListParams = useMemo(
    () => ({ page, pageSize: pageSize as ArsipPageSize, sort, order }),
    [page, pageSize, sort, order],
  )

  const query = useArsip(params)

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
    caption: 'Arsip dokumen pegawai',
    judul: 'Judul',
    jenisDokumen: 'Jenis dokumen',
    fileSize: 'Ukuran berkas',
    berlakuSampai: 'Berlaku sampai',
    createdAt: 'Dibuat',
    emptyTitle: 'Belum ada arsip dokumen',
    emptyHint: 'Unggah berkas untuk mulai.',
    loadingLabel: t('common.loadingTable'),
    resultLabel: `Menampilkan ${from}–${to} dari ${totalItems}`,
    pageInfo: t('common.pageInfo'),
    prevPage: t('common.prevPage'),
    nextPage: t('common.nextPage'),
    pageSizeLabel: t('common.pageSize'),
    itemsPerPage: t('common.itemsPerPage'),
    sortBy: t('common.sortBy'),
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">Kepegawaian</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        Arsip dokumen
      </h1>
      <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        Daftar arsip dokumen pegawai (KTP, KK, ijazah, SK, dan lainnya).
      </p>

      {query.isPending ? (
        <div
          role="status"
          aria-live="polite"
          className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center"
        >
          <LoadingBars size="md" decorative />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('common.loadingTable')}
          </p>
        </div>
      ) : query.isError || !data ? (
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            Gagal memuat data
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            Coba lagi beberapa saat.
          </p>
          <RetryButton label={t('common.retry')} onClick={() => void query.refetch()} />
        </div>
      ) : data.items.length === 0 ? (
        <div className="island-shell max-w-2xl rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {labels.emptyTitle}
          </h2>
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {labels.emptyHint}
          </p>
        </div>
      ) : (
        <ArsipTable
          items={data.items}
          page={page}
          pageSize={pageSize as ArsipPageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isFetching={query.isFetching}
          sort={sort}
          order={order}
          onPageChange={controls.setPage}
          onPageSizeChange={(size) => controls.setPageSize(size as ArsipPageSize)}
          onSortChange={(field, nextOrder) =>
            controls.applySort(field as ArsipSortField, nextOrder)
          }
          labels={labels}
        />
      )}
    </div>
  )
}
