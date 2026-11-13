import { Link } from '@tanstack/react-router'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'

const LEGAL_LINKS: readonly {
  to: '/privacy' | '/terms' | '/terms-of-reference'
  key: MessageKey
}[] = [
  { to: '/privacy', key: 'nav.privacy' },
  { to: '/terms', key: 'nav.terms' },
  { to: '/terms-of-reference', key: 'nav.reference' },
]

export default function Footer() {
  const t = useMessages()
  const year = new Date().getFullYear()

  return (
    <footer className="site-footer mt-16 px-4 pb-10 pt-8 text-[var(--sea-ink-soft)]">
      <div className="page-wrap flex flex-col gap-4 text-sm">
        <nav
          aria-label={t('footer.legalLabel')}
          className="flex flex-wrap items-center gap-x-5 gap-y-2"
        >
          {LEGAL_LINKS.map((item) => (
            <Link key={item.to} to={item.to} className="nav-link">
              {t(item.key)}
            </Link>
          ))}
        </nav>
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
          <p className="m-0">
            &copy; {year} {t('app.name')}. {t('app.description')}.
          </p>
          <p className="m-0">{t('footer.note')}</p>
        </div>
      </div>
    </footer>
  )
}
