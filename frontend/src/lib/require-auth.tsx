import { Navigate } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import { useMessages } from '@/i18n/locale-context'
import { useSession } from '@/lib/session'

function SessionLoading() {
  const t = useMessages()

  return (
    <div className="flex min-h-[60vh] items-center justify-center px-4">
      <div
        role="status"
        aria-live="polite"
        className="flex flex-col items-center gap-3 text-sm text-[var(--sea-ink-soft)]"
      >
        <LoadingBars size="md" decorative />
        <span>{t('session.loading')}</span>
      </div>
    </div>
  )
}

function SessionError({ onRetry }: { onRetry: () => void }) {
  const t = useMessages()

  return (
    <div className="flex min-h-[60vh] items-center justify-center px-4">
      <div className="island-shell w-full max-w-md rounded-2xl p-6 text-center">
        <h1 className="display-title mb-2 text-2xl font-bold text-[var(--sea-ink)]">
          {t('session.errorTitle')}
        </h1>
        <p className="mb-5 text-sm text-[var(--sea-ink-soft)]">
          {t('session.errorDescription')}
        </p>
        <RetryButton label={t('common.retry')} onClick={onRetry} />
      </div>
    </div>
  )
}

/**
 * Penjaga rute untuk halaman yang butuh sesi aktif. Penjaga ini hanya
 * membantu navigasi; backend tetap menjadi satu-satunya penentu hak akses.
 */
export function RequireAuth({ children }: { children: ReactNode }) {
  const session = useSession()

  if (session.isPending) {
    return <SessionLoading />
  }

  if (session.isError) {
    return <SessionError onRetry={() => void session.refetch()} />
  }

  if (!session.data) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

/** Penjaga rute untuk halaman masuk: pengguna yang sudah masuk diarahkan ke ruang kerja. */
export function RequireNoAuth({ children }: { children: ReactNode }) {
  const session = useSession()

  if (session.isPending) {
    return <SessionLoading />
  }

  if (session.isError) {
    return <SessionError onRetry={() => void session.refetch()} />
  }

  if (session.data) {
    return <Navigate to="/workspace" replace />
  }

  return <>{children}</>
}
