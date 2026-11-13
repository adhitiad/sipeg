import { Link } from '@tanstack/react-router'
import Header from '@/components/Header'
import Footer from '@/components/Footer'
import { Button } from '@/components/ui/button'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'

export interface LegalSection {
  title: MessageKey
  body: MessageKey
}

interface LegalPageProps {
  title: MessageKey
  lead?: MessageKey
  sections: readonly LegalSection[]
}

export default function LegalPage({ title, lead, sections }: LegalPageProps) {
  const t = useMessages()

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        <section className="page-wrap px-4 py-12 sm:py-16">
          <div className="island-shell rise-in rounded-2xl p-6 sm:p-10">
            <h1 className="display-title mb-3 max-w-3xl text-3xl leading-tight font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
              {t(title)}
            </h1>
            <p className="m-0 text-sm font-semibold text-[var(--kicker)]">
              {t('legal.draftNotice')}
            </p>
            {lead ? (
              <p className="mt-4 max-w-3xl leading-relaxed text-[var(--sea-ink-soft)]">
                {t(lead)}
              </p>
            ) : null}

            <div className="mt-8 grid gap-6">
              {sections.map((section) => (
                <section key={section.title}>
                  <h2 className="mb-1.5 text-lg font-semibold text-[var(--sea-ink)]">
                    {t(section.title)}
                  </h2>
                  <p className="m-0 max-w-3xl leading-relaxed text-[var(--sea-ink-soft)]">
                    {t(section.body)}
                  </p>
                </section>
              ))}
            </div>

            <p className="mt-8 text-xs text-[var(--sea-ink-soft)]">
              {t('legal.updated')} · {t('legal.contact')}
            </p>
            <div className="mt-6">
              <Button asChild variant="outline">
                <Link to="/">{t('common.backHome')}</Link>
              </Button>
            </div>
          </div>
        </section>
      </main>
      <Footer />
    </div>
  )
}
