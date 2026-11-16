import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Calculator, Receipt } from 'lucide-react'
import { toast } from 'sonner'
import LoadingBars from '@/components/LoadingBars'
import PayrollSlipsTable, { type PayrollPageSize } from '@/components/PayrollSlipsTable'
import RetryButton from '@/components/RetryButton'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useMessages } from '@/i18n/locale-context'
import {
  PAYROLL_SORT_FIELDS,
  generateSlip,
  usePayrollSlips,
  type PayrollListParams,
  type PayrollSortField,
} from '@/lib/payroll'
import { useSession } from '@/lib/session'
import { useTableControls } from '@/lib/table-state'

const SEARCH_DEBOUNCE_MS = 300

/**
 * Halaman slip gaji. Penyaringan, pengurutan, dan pemotongan baris dikerjakan
 * backend, jadi halaman ini menyimpan state tabel dan meneruskannya ke kueri.
 */
export default function PayrollSlipsPage() {
  const t = useMessages()
  const session = useSession()
  const permissions = session.data?.permissions ?? []
  const canRead = permissions.some((permission) => permission.startsWith('payroll:read:'))
  const canManage = permissions.includes('payroll:manage')
  const queryClient = useQueryClient()
  const [isGenerateOpen, setIsGenerateOpen] = useState(false)
  const [generateError, setGenerateError] = useState<string | null>(null)

  const generateMutation = useMutation({
    mutationFn: generateSlip,
    onSuccess: () => {
      toast.success(t('payroll.generateSuccess'))
      setIsGenerateOpen(false)
      setGenerateError(null)
      void queryClient.invalidateQueries({ queryKey: ['payroll', 'slips'] })
    },
    onError: (error: Error) => {
      setGenerateError(error.message)
    },
  })

  const [searchInput, setSearchInput] = useState('')
  const [periode, setPeriode] = useState('')
  const [status, setStatus] = useState('')
  const controls = useTableControls<PayrollSortField>({
    sortFields: PAYROLL_SORT_FIELDS,
    initialSort: 'periode',
    initialOrder: 'desc',
    initialPageSize: 10,
  })
  const { sort, order, page, pageSize, resetPage } = controls

  useEffect(() => {
    const timer = setTimeout(() => {
      setPeriode(searchInput.trim())
      resetPage()
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [searchInput, resetPage])

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
        <div className="mb-2 flex flex-wrap items-end justify-between gap-3">
          <div>
            <p className="island-kicker mb-2">{t('payroll.kicker')}</p>
            <h1 className="display-title mb-2 text-3xl font-bold text-[var(--sea-ink)] sm:text-4xl">
              {t('payroll.slipsTitle')}
            </h1>
          </div>
          {canManage ? (
            <Button type="button" onClick={() => setIsGenerateOpen(true)}>
              <Calculator className="mr-1.5 h-4 w-4" aria-hidden="true" />
              {t('payroll.generateButton')}
            </Button>
          ) : null}
        </div>
        <p className="m-0 mb-6 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)]">
          {t('payroll.slipsLead')}
        </p>

        <PayrollSlipsTable
          slips={data.items}
          page={page}
          pageSize={pageSize as PayrollPageSize}
          totalPages={data.totalPages}
          totalItems={data.totalItems}
          isFetching={query.isFetching}
          search={searchInput}
          status={status || null}
          sort={sort}
          order={order}
          onPageChange={controls.setPage}
          onPageSizeChange={(size) => controls.setPageSize(size)}
          onSearchChange={(value) => setSearchInput(value)}
          onStatusChange={(value) => {
            setStatus(value ?? '')
            resetPage()
          }}
          onSortChange={(field, next) => controls.applySort(field, next)}
        />

        <Dialog
          open={isGenerateOpen}
          onOpenChange={(open) => {
            setIsGenerateOpen(open)
            if (!open) setGenerateError(null)
          }}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t('payroll.generateDialogTitle')}</DialogTitle>
              <DialogDescription>{t('payroll.generateDialogLead')}</DialogDescription>
            </DialogHeader>
            <form
              className="grid gap-4"
              noValidate
              onSubmit={(event) => {
                event.preventDefault()
                const data = Object.fromEntries(
                  new FormData(event.currentTarget).entries(),
                ) as Record<string, string>
                const tingkat = data.tingkat?.trim() ? Number(data.tingkat) : undefined
                generateMutation.mutate({
                  userId: data.userId?.trim() ?? '',
                  periode: data.periode ?? '',
                  kategoriStatusId: data.kategoriStatusId?.trim() || undefined,
                  tingkat,
                })
              }}
            >
              <div className="grid gap-1.5">
                <Label htmlFor="generate-userId">{t('payroll.generateUserId')}</Label>
                <Input
                  id="generate-userId"
                  name="userId"
                  required
                  className="h-11 font-mono text-xs"
                  placeholder="00000000-0000-4000-8000-..."
                  autoComplete="off"
                />
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="grid gap-1.5">
                  <Label htmlFor="generate-periode">{t('payroll.generatePeriode')}</Label>
                  <Input
                    id="generate-periode"
                    name="periode"
                    type="month"
                    required
                    className="h-11"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="generate-tingkat">{t('payroll.generateTingkat')}</Label>
                  <Input
                    id="generate-tingkat"
                    name="tingkat"
                    type="number"
                    min={1}
                    step={1}
                    placeholder="1"
                    className="h-11"
                  />
                </div>
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="generate-kategoriStatusId">
                  {t('payroll.generateKategori')}
                </Label>
                <Input
                  id="generate-kategoriStatusId"
                  name="kategoriStatusId"
                  className="h-11 font-mono text-xs"
                  placeholder="d0000000-0000-0000-0000-..."
                  autoComplete="off"
                />
              </div>
              {generateError ? (
                <p role="alert" className="m-0 text-sm text-destructive">
                  {generateError}
                </p>
              ) : null}
              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setIsGenerateOpen(false)}
                >
                  {t('common.cancel')}
                </Button>
                <Button type="submit" disabled={generateMutation.isPending}>
                  {generateMutation.isPending
                    ? t('payroll.generating')
                    : t('payroll.generateButton')}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </section>
    </div>
  )
}
