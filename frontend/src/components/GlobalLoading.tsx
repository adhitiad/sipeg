import LoadingBars from '@/components/LoadingBars'
import { useMessages } from '@/i18n/locale-context'

export default function GlobalLoading() {
  const t = useMessages()

  return (
    <main className="flex min-h-[50vh] items-center justify-center px-4 py-16">
      <div
        role="status"
        aria-live="polite"
        className="flex flex-col items-center gap-3 text-sm text-[var(--sea-ink-soft)]"
      >
        <LoadingBars size="md" decorative />
        <span>{t('state.loading.title')}</span>
      </div>
    </main>
  )
}
