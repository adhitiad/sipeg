import { Link, createFileRoute } from '@tanstack/react-router'
import Header from '@/components/Header'
import Footer from '@/components/Footer'
import { Button } from '@/components/ui/button'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'

export const Route = createFileRoute('/about')({
  component: AboutPage,
})

const SECTIONS: readonly { title: MessageKey; body: MessageKey }[] = [
  { title: 'about.whatTitle', body: 'about.whatBody' },
  { title: 'about.whoTitle', body: 'about.whoBody' },
  { title: 'about.statusTitle', body: 'about.statusBody' },
  { title: 'about.accessTitle', body: 'about.accessBody' },
]

function AboutPage() {
  const t = useMessages()

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        <section className="page-wrap px-4 py-12 sm:py-16">
          <div className="island-shell rise-in rounded-2xl p-6 sm:p-10">
            <h1 className="display-title mb-4 max-w-3xl text-3xl leading-tight font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
              {t('about.title')}
            </h1>
            <p className="m-0 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)] sm:text-lg">
              {t('about.lead')}
            </p>

            <div className="mt-8 grid gap-6">
              {SECTIONS.map((section) => (
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

            <div className="mt-8">
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
