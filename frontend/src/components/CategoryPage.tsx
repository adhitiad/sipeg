import { useEffect, useMemo, useState } from 'react'
import { ShieldAlert } from 'lucide-react'
import CategoryTable, { type CategoryPageSize } from '@/components/CategoryTable'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import { useLocale, useMessages } from '@/i18n/locale-context'
import {
  CATEGORY_SORT_FIELDS,
  useCategories,
  type CategoryListParams,
  type CategorySortField,
} from '@/lib/audit'
import { useSession } from '@/lib/session'
import { useTableControls, visibleRange } from '@/lib/table-state'

const SEARCH_DEBOUNCE_MS = 300

/**
 * Halaman daftar kategori pegawai. Backend memegang pagination, sorting, dan
 * filtering, jadi halaman ini hanya menyimpan state kendali tabel.
 */
export default function CategoryPage() {
  const t = useMessages()
  const { locale } = useLocale()
  const session = useSession()

  const permissions = session.data?.permissions ?? []
  const canRead = permissions.includes('category:read')

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const controls = useTableControls<CategorySortField>({
    sortFields: CATEGORY_SORT_FIELDS,
    initialSort: 'kode',
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

  const params: CategoryListParams = useMemo(
    () => ({ page, pageSize, search, sort, order }),
    [page, pageSize, search, sort, order],
  )

  const query = useCategories(params)

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
    caption: t('categories.tableCaption'),
    code: t('categories.colCode'),
    name: t('categories.colName'),
    retirementAge: t('categories.colRetirementAge'),
    status: t('organizations.colStatus'),
    active: t('organizations.statusActive'),
    inactive: t('organizations.statusInactive'),
    searchLabel: t('categories.searchLabel'),
    searchPlaceholder: t('categories.searchPlaceholder'),
    emptyTitle: t('categories.emptyTitle'),
    emptyHint: t('categories.emptyDescription'),
    loadingLabel: t('common.loadingTable'),
    resultLabel: t('categories.resultCount', {
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
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('categories.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('categories.title')}
      </h1>
      <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('categories.lead')}
      </p>

      {query.isPending ? (
        <div
          role="status"
          aria-live="polite"
          className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center"
        >
          <LoadingBars size="md" decorative />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('categories.loadingTitle')}
          </p>
        </div>
      ) : query.isError || !data ? (
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('categories.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('categories.errorDescription')}
          </p>
          <RetryButton label={t('common.retry')} onClick={() => void query.refetch()} />
        </div>
      ) : data.items.length === 0 ? (
        <div className="island-shell max-w-2xl rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('categories.emptyTitle')}
          </h2>
          <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('categories.emptyDescription')}
          </p>
        </div>
      ) : (
        <CategoryTable
          categories={data.items}
          page={page}
          pageSize={pageSize as CategoryPageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isLoading={query.isPending}
          isFetching={query.isFetching}
          search={searchInput}
          sort={sort}
          order={order}
          onPageChange={controls.setPage}
          onPageSizeChange={(size) => controls.setPageSize(size as CategoryPageSize)}
          onSearchChange={setSearchInput}
          onSortChange={controls.applySort}
          labels={labels}
        />
      )}
    </div>
  )
}
