import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from '@tanstack/react-router'
import { ArrowLeft } from 'lucide-react'
import { toast } from 'sonner'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useMessages } from '@/i18n/locale-context'
import { ApiError } from '@/lib/api'
import { confirmPayrollPayment, payrollSlipQueryKey, usePayrollSlip } from '@/lib/payroll'
import { useSession } from '@/lib/session'

export default function PayrollSlipDetailPage() {
  const t = useMessages()
  const { payrollId } = useParams({ from: '/_app/payroll/slips/$payrollId' })
  const session = useSession()
  const queryClient = useQueryClient()
  const permissions = session.data?.permissions ?? []
  const canManage = permissions.includes('payroll:manage')

  const query = usePayrollSlip(payrollId)
  const [approver, setApprover] = useState('')
  const [paymentDate, setPaymentDate] = useState('')
  const [reference, setReference] = useState('')
  const [note, setNote] = useState('')

  const mutation = useMutation({
    mutationFn: () =>
      confirmPayrollPayment(payrollId, {
        pelakuPengesahan: approver.trim(),
        tanggalBayar: paymentDate,
        referensiTransaksi: reference.trim() || null,
        keterangan: note.trim() || null,
      }),
    onSuccess: () => {
      toast.success(t('payroll.paymentSuccess'))
      void queryClient.invalidateQueries({ queryKey: payrollSlipQueryKey(payrollId) })
      void queryClient.invalidateQueries({ queryKey: ['payroll', 'slips'] })
    },
    onError: (error) => {
      if (error instanceof ApiError && error.status === 409) {
        toast.error(t('payroll.paymentConflict'))
        return
      }
      if (error instanceof ApiError && (error.status === 400 || error.status === 422)) {
        toast.error(t('payroll.paymentValidation'))
        return
      }
      toast.error(t('payroll.paymentError'))
    },
  })

  if (query.isPending) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <LoadingBars label={t('payroll.detailLoadingTitle')} />
          <h1 className="display-title mt-3 text-2xl font-bold text-[var(--sea-ink)]">
            {t('payroll.detailLoadingTitle')}
          </h1>
        </section>
      </div>
    )
  }

  if (query.isError || !query.data) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <h1 className="display-title mb-2 text-2xl font-bold text-[var(--sea-ink)]">
            {t('payroll.detailErrorTitle')}
          </h1>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('payroll.detailErrorDescription')}
          </p>
          <div className="mt-4 flex justify-center gap-3">
            <RetryButton label={t('common.retry')} onClick={() => query.refetch()} />
            <Link
              to="/payroll/slips"
              className="inline-flex h-10 items-center rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-4 text-sm font-semibold text-[var(--sea-ink)]"
            >
              {t('payroll.backToList')}
            </Link>
          </div>
        </section>
      </div>
    )
  }

  const slip = query.data
  const canConfirm = canManage && slip.status === 'TERBIT'

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <section className="island-shell rise-in rounded-2xl p-6 sm:p-8">
        <Link
          to="/payroll/slips"
          className="mb-4 inline-flex items-center gap-2 text-sm font-medium text-[var(--sea-ink-soft)] hover:text-[var(--sea-ink)]"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          {t('payroll.backToList')}
        </Link>
        <p className="island-kicker mb-2">{t('payroll.kicker')}</p>
        <h1 className="display-title mb-2 text-3xl font-bold text-[var(--sea-ink)] sm:text-4xl">
          {t('payroll.detailTitle')}
        </h1>
        <p className="m-0 mb-6 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)]">
          {t('payroll.detailLead')}
        </p>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div className="island-shell rounded-xl p-4">
            <p className="island-kicker mb-1">{t('payroll.colSlipNumber')}</p>
            <p className="m-0 text-base font-semibold text-[var(--sea-ink)]">
              {slip.nomorSlip}
            </p>
          </div>
          <div className="island-shell rounded-xl p-4">
            <p className="island-kicker mb-1">{t('payroll.colPeriod')}</p>
            <p className="m-0 text-base font-semibold text-[var(--sea-ink)]">
              {slip.periodeLabel ?? slip.periode}
            </p>
          </div>
          <div className="island-shell rounded-xl p-4">
            <p className="island-kicker mb-1">{t('payroll.colEmployee')}</p>
            <p className="m-0 text-base font-semibold text-[var(--sea-ink)]">
              {slip.nama ?? '-'}
            </p>
          </div>
          <div className="island-shell rounded-xl p-4">
            <p className="island-kicker mb-1">{t('payroll.colStatus')}</p>
            <p className="m-0 text-base font-semibold text-[var(--sea-ink)]">
              {slip.statusLabel ?? slip.status}
            </p>
          </div>
          <div className="island-shell rounded-xl p-4">
            <p className="island-kicker mb-1">{t('payroll.colPayDate')}</p>
            <p className="m-0 text-base font-semibold text-[var(--sea-ink)]">
              {slip.tanggalGaji}
            </p>
          </div>
          <div className="island-shell rounded-xl p-4">
            <p className="island-kicker mb-1">{t('payroll.colPaidAt')}</p>
            <p className="m-0 text-base font-semibold text-[var(--sea-ink)]">
              {slip.tanggalDibayar ?? '-'}
            </p>
          </div>
        </div>

        <div className="mt-6 grid gap-6 lg:grid-cols-3">
          <div className="island-shell rounded-xl p-4">
            <h2 className="m-0 mb-3 text-lg font-semibold text-[var(--sea-ink)]">
              {t('payroll.baseSalary')}
            </h2>
            <p className="m-0 text-xl font-bold text-[var(--sea-ink)]">
              {slip.gajiPokok}
            </p>
          </div>
          <div className="island-shell rounded-xl p-4">
            <h2 className="m-0 mb-3 text-lg font-semibold text-[var(--sea-ink)]">
              {t('payroll.allowances')}
            </h2>
            {(slip.pendapatan?.length ?? 0) === 0 ? (
              <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
                {t('payroll.noItems')}
              </p>
            ) : (
              <ul className="m-0 grid list-none gap-2 p-0">
                {slip.pendapatan?.map((item, index) => (
                  <li
                    key={`${item.jenis}-${index}`}
                    className="flex justify-between gap-3 text-sm"
                  >
                    <span className="text-[var(--sea-ink-soft)]">{item.jenis}</span>
                    <span className="font-medium text-[var(--sea-ink)]">
                      {item.nilai}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
          <div className="island-shell rounded-xl p-4">
            <h2 className="m-0 mb-3 text-lg font-semibold text-[var(--sea-ink)]">
              {t('payroll.deductions')}
            </h2>
            {(slip.potongan?.length ?? 0) === 0 ? (
              <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
                {t('payroll.noItems')}
              </p>
            ) : (
              <ul className="m-0 grid list-none gap-2 p-0">
                {slip.potongan?.map((item, index) => (
                  <li
                    key={`${item.jenis}-${index}`}
                    className="flex justify-between gap-3 text-sm"
                  >
                    <span className="text-[var(--sea-ink-soft)]">{item.jenis}</span>
                    <span className="font-medium text-[var(--sea-ink)]">
                      {item.nominal}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>

        <div className="island-shell mt-6 rounded-xl p-4">
          <div className="flex items-baseline justify-between gap-3">
            <h2 className="m-0 text-lg font-semibold text-[var(--sea-ink)]">
              {t('payroll.total')}
            </h2>
            <p className="m-0 text-2xl font-bold text-[var(--sea-ink)]">
              {slip.totalGaji}
            </p>
          </div>
        </div>

        {canConfirm ? (
          <div className="island-shell mt-6 rounded-xl p-4">
            <h2 className="m-0 mb-1 text-lg font-semibold text-[var(--sea-ink)]">
              {t('payroll.paymentTitle')}
            </h2>
            <p className="m-0 mb-4 text-sm text-[var(--sea-ink-soft)]">
              {t('payroll.paymentDescription')}
            </p>
            <form
              className="grid gap-3 sm:grid-cols-2"
              onSubmit={(event) => {
                event.preventDefault()
                mutation.mutate()
              }}
            >
              <label className="grid gap-1">
                <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
                  {t('payroll.paymentApprover')}
                </span>
                <Input
                  value={approver}
                  onChange={(event) => setApprover(event.target.value)}
                  required
                  className="h-11"
                />
              </label>
              <label className="grid gap-1">
                <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
                  {t('payroll.paymentDate')}
                </span>
                <Input
                  type="date"
                  value={paymentDate}
                  onChange={(event) => setPaymentDate(event.target.value)}
                  required
                  className="h-11"
                />
              </label>
              <label className="grid gap-1 sm:col-span-2">
                <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
                  {t('payroll.paymentReference')}
                </span>
                <Input
                  value={reference}
                  onChange={(event) => setReference(event.target.value)}
                  className="h-11"
                />
              </label>
              <label className="grid gap-1 sm:col-span-2">
                <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
                  {t('payroll.paymentNote')}
                </span>
                <Textarea
                  value={note}
                  onChange={(event) => setNote(event.target.value)}
                  rows={3}
                />
              </label>
              <div className="sm:col-span-2">
                <Button type="submit" disabled={mutation.isPending}>
                  {mutation.isPending
                    ? t('payroll.confirming')
                    : t('payroll.confirmPayment')}
                </Button>
              </div>
            </form>
          </div>
        ) : null}
      </section>
    </div>
  )
}
