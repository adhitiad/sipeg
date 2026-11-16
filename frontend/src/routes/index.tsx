import { Link, createFileRoute } from '@tanstack/react-router'
import Header from '@/components/Header'
import Footer from '@/components/Footer'
import { Button } from '@/components/ui/button'
import { useMessages } from '@/i18n/locale-context'
import { DEFAULT_LOCALE, translate, type MessageKey } from '@/i18n/dictionary'
import { socialMeta } from '@/lib/seo'

const t = (key: MessageKey) => translate(DEFAULT_LOCALE, key)

export const Route = createFileRoute('/')({
  component: LandingPage,
  head: () => ({
    meta: [
      { title: t('seo.landingTitle') },
      { name: 'description', content: t('seo.landingDescription') },
      ...socialMeta({
        title: t('seo.landingTitle'),
        description: t('seo.landingDescription'),
        path: '/',
      }),
    ],
  }),
})

const SCOPE_CARDS: readonly { title: MessageKey; body: MessageKey }[] = [
  { title: 'landing.scope.employee.title', body: 'landing.scope.employee.body' },
  { title: 'landing.scope.organization.title', body: 'landing.scope.organization.body' },
  { title: 'landing.scope.attendance.title', body: 'landing.scope.attendance.body' },
  { title: 'landing.scope.leave.title', body: 'landing.scope.leave.body' },
  { title: 'landing.scope.payroll.title', body: 'landing.scope.payroll.body' },
  { title: 'landing.scope.performance.title', body: 'landing.scope.performance.body' },
]

const ROLE_CARDS: readonly { title: MessageKey; body: MessageKey }[] = [
  { title: 'landing.role.admin.title', body: 'landing.role.admin.body' },
  { title: 'landing.role.operator.title', body: 'landing.role.operator.body' },
  { title: 'landing.role.atasan.title', body: 'landing.role.atasan.body' },
  { title: 'landing.role.pegawai.title', body: 'landing.role.pegawai.body' },
]

const ACCESS_CARDS: readonly { title: MessageKey; body: MessageKey }[] = [
  { title: 'landing.access.account.title', body: 'landing.access.account.body' },
  { title: 'landing.access.role.title', body: 'landing.access.role.body' },
  { title: 'landing.access.audit.title', body: 'landing.access.audit.body' },
]

function LandingPage() {
  const t = useMessages()

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        <section className="page-wrap px-4 py-14 sm:py-20">
          <div className="island-shell rise-in rounded-2xl p-6 sm:p-10">
            <h1 className="display-title mb-4 max-w-3xl text-3xl leading-[1.15] font-semibold tracking-tight text-[var(--sea-ink)] sm:text-5xl">
              {t('landing.heroTitle')}
            </h1>
            <p className="mb-8 max-w-2xl text-base leading-relaxed text-[var(--sea-ink-soft)] sm:text-lg">
              {t('landing.heroLead')}
            </p>
            <div className="flex flex-col gap-3 sm:flex-row">
              <Button asChild size="lg" className="h-12 min-w-44 text-sm">
                <Link to="/login">{t('landing.primaryCta')}</Link>
              </Button>
              <Button
                asChild
                variant="outline"
                size="lg"
                className="h-12 min-w-44 text-sm"
              >
                <Link to="/about">{t('landing.secondaryCta')}</Link>
              </Button>
            </div>
          </div>
        </section>

        <section className="page-wrap px-4 pb-14 sm:pb-20">
          <h2 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-3xl">
            {t('landing.scopeTitle')}
          </h2>
          <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)] sm:text-base">
            {t('landing.scopeLead')}
          </p>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {SCOPE_CARDS.map((card) => (
              <article
                key={card.title}
                className="feature-card rounded-2xl border border-[var(--line)] p-5"
              >
                <h3 className="mb-1.5 text-base font-semibold text-[var(--sea-ink)]">
                  {t(card.title)}
                </h3>
                <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
                  {t(card.body)}
                </p>
              </article>
            ))}
          </div>
        </section>

        <section className="page-wrap px-4 pb-14 sm:pb-20">
          <h2 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-3xl">
            {t('landing.rolesTitle')}
          </h2>
          <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)] sm:text-base">
            {t('landing.rolesLead')}
          </p>
          <div className="grid gap-4 sm:grid-cols-2">
            {ROLE_CARDS.map((card) => (
              <article
                key={card.title}
                className="feature-card rounded-2xl border border-[var(--line)] p-5"
              >
                <h3 className="mb-1.5 text-base font-semibold text-[var(--sea-ink)]">
                  {t(card.title)}
                </h3>
                <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
                  {t(card.body)}
                </p>
              </article>
            ))}
          </div>
        </section>

        <section className="page-wrap px-4 pb-16 sm:pb-20">
          <h2 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-3xl">
            {t('landing.accessTitle')}
          </h2>
          <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)] sm:text-base">
            {t('landing.accessLead')}
          </p>
          <div className="grid gap-4">
            {ACCESS_CARDS.map((card) => (
              <article
                key={card.title}
                className="feature-card rounded-2xl border border-[var(--line)] p-5 sm:flex sm:items-baseline sm:gap-6"
              >
                <h3 className="mb-1.5 text-base font-semibold text-[var(--sea-ink)] sm:mb-0 sm:min-w-56">
                  {t(card.title)}
                </h3>
                <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
                  {t(card.body)}
                </p>
              </article>
            ))}
          </div>
        </section>

        <section className="page-wrap px-4 pb-20 text-center">
          <p className="mx-auto mb-2 max-w-lg text-lg font-semibold text-[var(--sea-ink)]">
            {t('landing.closeTitle')}
          </p>
          <p className="mx-auto mb-6 max-w-lg text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('landing.closeBody')}
          </p>
          <Button asChild size="lg" className="h-12 min-w-48 text-sm">
            <Link to="/login">{t('landing.closeCta')}</Link>
          </Button>
        </section>
      </main>
      <Footer />
    </div>
  )
}
