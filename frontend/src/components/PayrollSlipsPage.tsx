import { useEffect, useMemo, useState } from 'react'
import { Receipt } from 'lucide-react'
import LoadingBars from '@/components/LoadingBars'
import PayrollSlipsTable from '@/components/PayrollSlipsTable'
import RetryButton from '@/components/RetryButton'
import { useMessages } from '@/i18n/locale-context'
import {
  usePayrollSlips,
  type PayrollListParams,
  type PayrollSortField,
  type PayrollSortOrder,
} from '@/lib/payroll'
import { useSession } from '@/lib/session'

const SEARCH_DEBOUNCE_MS = 300

export default function PayrollSlipsPage() {
  const t = useMessages()
  const session = useSession()
  const permissions = session.data?.permissions ?? []
  const canRead = permissions.some((permission) => permission.startsWith('payroll:read:'))

  const [searchInput, setSearchInput] = useState('')
  const [periode, setPeriode] = useState('')
  const [status, setStatus] = useState('')
  const [sort, setSort] = useState<PayrollSortField>('periode')
  const [order, setOrder] = useState<PayrollSortOrder>('desc')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState<10 | 25 | 50 | 100>(10)

  useEffect(() => {
    const timer = setTimeout(() => {
      setPeriode(searchInput.trim())
      setPage(1)
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [searchInput])

  const params: PayrollListParams = useMemo(
    () => ({ page, pageSize, periode, status, sort, order }),
    [page, pageSize, periode, status, sort, order],
  )

  const query = usePayrollSlips(params)

  if (session.data && !canRead) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <Receipt
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

  if (query.isPending) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <LoadingBars label={t('payroll.slipsLoadingTitle')} />
          <h1 className="display-title mt-3 text-2xl font-bold text-[var(--sea-ink)]">
            {t('payroll.slipsLoadingTitle')}
          </h1>
        </section>
      </div>
    )
  }

  if (query.isError) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <h1 className="display-title mb-2 text-2xl font-bold text-[var(--sea-ink)]">
            {t('payroll.slipsErrorTitle')}
          </h1>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('payroll.slipsErrorDescription')}
          </p>
          <div className="mt-4 flex justify-center">
            <RetryButton label={t('common.retry')} onClick={() => query.refetch()} />
          </div>
        </section>
      </div>
    )
  }

  const data = query.data

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <section className="island-shell rise-in rounded-2xl p-6 sm:p-8">
        <p className="island-kicker mb-2">{t('payroll.kicker')}</p>
        <h1 className="display-title mb-2 text-3xl font-bold text-[var(--sea-ink)] sm:text-4xl">
          {t('payroll.slipsTitle')}
        </h1>
        <p className="m-0 mb-6 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)]">
          {t('payroll.slipsLead')}
        </p>

        <PayrollSlipsTable
          slips={data.items}
          page={data.page}
          pageSize={pageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isFetching={query.isFetching}
          search={searchInput}
          status={status || null}
          sort={sort}
          order={order}
          onPageChange={setPage}
          onPageSizeChange={(size) => {
            setPageSize(size)
            setPage(1)
          }}
          onSearchChange={(value) => setSearchInput(value)}
          onStatusChange={(value) => {
            setStatus(value ?? '')
            setPage(1)
          }}
          onSortChange={(field) => {
            setSort(field)
            setOrder((prev) => (sort === field && prev === 'asc' ? 'desc' : 'asc'))
            setPage(1)
          }}
        />
      </section>
    </div>
  )
}
