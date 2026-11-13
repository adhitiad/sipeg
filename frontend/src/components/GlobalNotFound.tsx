import { Link } from '@tanstack/react-router'
import { FileQuestion } from 'lucide-react'
import { useMessages } from '@/i18n/locale-context'
import { Button } from '@/components/ui/button'

export default function GlobalNotFound() {
  const t = useMessages()

  return (
    <main className="flex min-h-[60vh] items-center justify-center px-4 py-16">
      <div className="island-shell w-full max-w-md rounded-2xl p-6 text-center">
        <FileQuestion
          className="mx-auto mb-4 h-8 w-8 text-[var(--sea-ink-soft)]"
          aria-hidden="true"
        />
        <h1 className="display-title mb-2 text-2xl leading-tight font-semibold text-[var(--sea-ink)]">
          {t('state.notFound.title')}
        </h1>
        <p className="mb-6 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
          {t('state.notFound.description')}
        </p>
        <Button asChild>
          <Link to="/">{t('common.backHome')}</Link>
        </Button>
      </div>
    </main>
  )
}
