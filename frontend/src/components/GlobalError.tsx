import { TriangleAlert } from 'lucide-react'
import { useMessages } from '@/i18n/locale-context'
import RetryButton from '@/components/RetryButton'

export default function GlobalError({ error }: { error: unknown }) {
  const t = useMessages()
  const detail = error instanceof Error ? error.message : String(error ?? '')

  return (
    <main className="flex min-h-[60vh] items-center justify-center px-4 py-16">
      <div className="island-shell w-full max-w-md rounded-2xl p-6 text-center">
        <TriangleAlert
          className="mx-auto mb-4 h-8 w-8 text-[var(--sea-ink-soft)]"
          aria-hidden="true"
        />
        <h1 className="display-title mb-2 text-2xl leading-tight font-semibold text-[var(--sea-ink)]">
          {t('state.error.title')}
        </h1>
        <p className="mb-6 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
          {t('state.error.description')}
        </p>
        {detail ? (
          <p className="mb-6 rounded-xl border border-[var(--line)] bg-[var(--chip-bg)] px-3 py-2 text-left text-xs break-words text-[var(--sea-ink-soft)]">
            {detail}
          </p>
        ) : null}
        <RetryButton label={t('common.retry')} onClick={() => window.location.reload()} />
      </div>
    </main>
  )
}
