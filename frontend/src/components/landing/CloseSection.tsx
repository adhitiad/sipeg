import { Link } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { useMessages } from '@/i18n/locale-context'

export default function CloseSection() {
  const t = useMessages()

  return (
    <section className="page-wrap px-4 pb-20 text-center sm:pb-28">
      <h2 className="display-title mx-auto mb-3 max-w-lg text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-3xl">
        {t('landing.closeTitle')}
      </h2>
      <p className="mx-auto mb-7 max-w-md text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('landing.closeBody')}
      </p>
      <Button asChild size="lg" className="h-12 min-w-48 text-sm">
        <Link to="/login">{t('landing.closeCta')}</Link>
      </Button>
    </section>
  )
}
