import { useParams } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { CheckCircle2, ShieldAlert } from 'lucide-react'
import Header from '@/components/Header'
import Footer from '@/components/Footer'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import { useMessages } from '@/i18n/locale-context'
import { fetchPublicSlipVerification } from '@/lib/slip-verify'

export default function SlipVerifyPage() {
  const t = useMessages()
  const { token } = useParams({ from: '/verify/slip/$token' })
  const query = useQuery({
    queryKey: ['slip-verify', token] as const,
    queryFn: () => fetchPublicSlipVerification(token),
    retry: false,
    staleTime: 60_000,
    enabled: token.length === 22,
  })

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex flex-1 items-center justify-center px-4 py-14">
        <section
          aria-labelledby="verify-title"
          className="island-shell mx-auto w-full max-w-md rounded-2xl p-6 text-center sm:p-8"
        >
          {token.length !== 22 ? (
            <InvalidToken message={t('verify.invalid')} />
          ) : query.isPending ? (
            <div className="py-6">
              <LoadingBars size="md" label={t('verify.title')} />
            </div>
          ) : query.isError ? (
            <InvalidToken
              message={t('verify.notFound')}
              onRetry={() => query.refetch()}
            />
          ) : (
            <Verified slip={query.data} />
          )}
        </section>
      </main>
      <Footer />
    </div>
  )
}

function Verified({
  slip,
}: {
  slip: {
    nama: string
    periodeLabel?: string
    periode: string
    statusLabel?: string
    status: string
  }
}) {
  const t = useMessages()
  return (
    <>
      <CheckCircle2
        className="mx-auto mb-3 h-10 w-10 text-[var(--lagoon)]"
        aria-hidden="true"
      />
      <p className="island-kicker mb-1">{t('verify.kicker')}</p>
      <h1
        id="verify-title"
        className="display-title mb-4 text-2xl font-semibold text-[var(--sea-ink)]"
      >
        {t('verify.validTitle')}
      </h1>
      <dl className="m-0 mb-5 grid gap-3 text-left">
        <div className="flex items-baseline justify-between gap-4 border-b border-[var(--line)] pb-3">
          <dt className="text-xs font-semibold uppercase tracking-[0.14em] text-[var(--kicker)]">
            {t('verify.fieldName')}
          </dt>
          <dd className="m-0 text-base font-semibold text-[var(--sea-ink)]">
            {slip.nama}
          </dd>
        </div>
        <div className="flex items-baseline justify-between gap-4 border-b border-[var(--line)] pb-3">
          <dt className="text-xs font-semibold uppercase tracking-[0.14em] text-[var(--kicker)]">
            {t('verify.fieldPeriod')}
          </dt>
          <dd className="m-0 text-base font-semibold text-[var(--sea-ink)]">
            {slip.periodeLabel ?? slip.periode}
          </dd>
        </div>
        <div className="flex items-baseline justify-between gap-4">
          <dt className="text-xs font-semibold uppercase tracking-[0.14em] text-[var(--kicker)]">
            {t('verify.fieldStatus')}
          </dt>
          <dd className="m-0 text-base font-semibold text-[var(--lagoon)]">
            {slip.statusLabel ?? slip.status}
          </dd>
        </div>
      </dl>
      <p className="m-0 text-xs leading-relaxed text-[var(--sea-ink-soft)]">
        {t('verify.disclaimer')}
      </p>
    </>
  )
}

function InvalidToken({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const t = useMessages()
  return (
    <>
      <ShieldAlert
        className="mx-auto mb-3 h-10 w-10 text-[var(--sea-ink-soft)]"
        aria-hidden="true"
      />
      <h1
        id="verify-title"
        className="display-title mb-3 text-2xl font-semibold text-[var(--sea-ink)]"
      >
        {t('verify.invalidTitle')}
      </h1>
      <p className="m-0 mb-4 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {message}
      </p>
      {onRetry ? <RetryButton label={t('common.retry')} onClick={onRetry} /> : null}
    </>
  )
}
