import { useEffect, useMemo, useState } from 'react'
import { ShieldAlert } from 'lucide-react'
import CategoryTable, { type CategoryPageSize } from '@/components/CategoryTable'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import { useLocale, useMessages } from '@/i18n/locale-context'
import {
  useCategories,
  type CategoryListParams,
  type CategorySortField,
  type CategorySortOrder,
} from '@/lib/audit'
import { useSession } from '@/lib/session'

const SEARCH_DEBOUNCE_MS = 300

export default function CategoryPage() {
  const t = useMessages()
  const { locale } = useLocale()
  const session = useSession()

  const permissions = session.data?.permissions ?? []
  const canRead = permissions.includes('category:read')

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [sort, setSort] = useState<CategorySortField>('kode')
  const [order, setOrder] = useState<CategorySortOrder>('asc')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState<CategoryPageSize>(25)

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput.trim())
      setPage(1)
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [searchInput])

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
  const from = totalItems === 0 ? 0 : (page - 1) * pageSize + 1
  const to = Math.min(page * pageSize, totalItems)

  const labels = {
    caption: t('categories.tableCaption'),
    code: t('categories.colCode'),
    name: t('categories.colName'),
    retirementAge: t('categories.colRetirementAge'),
    status: t('organizations.colStatus'),
    active: t('organizations.statusActive'),
    inactive: t('organizations.statusInactive'),
    noValue: t('audit.noValue'),
    resultLabel: t('categories.resultCount', {
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
    searchLabel: t('categories.searchLabel'),
    searchPlaceholder: t('categories.searchPlaceholder'),
    pageSizeLabel: t('organizations.pageSize'),
    itemsPerPage: t('categories.itemsPerPage', {
      from: new Intl.NumberFormat(localeTag).format(from),
      to: new Intl.NumberFormat(localeTag).format(to),
      total: new Intl.NumberFormat(localeTag).format(totalItems),
    }),
  }

  function handleSortChange(field: CategorySortField) {
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
      <p className="island-kicker mb-2">{t('categories.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('categories.title')}
      </h1>
      <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('categories.lead')}
      </p>

      {query.isPending ? (
        <div className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center">
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
          pageSize={pageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isFetching={query.isFetching}
          search={searchInput}
          sort={sort}
          order={order}
          onPageChange={setPage}
          onPageSizeChange={setPageSize}
          onSearchChange={setSearchInput}
          onSortChange={handleSortChange}
          labels={labels}
        />
      )}
    </div>
  )
}
