import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import ModulePanel from '@/components/landing/ModulePanel'
import { Button } from '@/components/ui/button'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'

/**
 * Tiga fakta produk di bawah tombol. Semuanya menggambarkan sistem yang
 * benar-benar ada, bukan angka rekaan atau klaim ketersediaan.
 */
const FACTS: readonly MessageKey[] = [
  'landing.heroFactRoles',
  'landing.heroFactAudit',
  'landing.heroFactStack',
]

export default function HeroSection() {
  const t = useMessages()

  return (
    <section className="page-wrap px-4 pt-12 pb-14 sm:pt-16 sm:pb-20">
      <div className="grid items-start gap-10 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] lg:gap-14">
        <div className="rise-in">
          <p className="island-kicker mb-4">{t('landing.kicker')}</p>
          <h1 className="display-title mb-5 max-w-xl text-3xl leading-[1.15] font-semibold tracking-tight text-[var(--sea-ink)] sm:text-5xl">
            {t('landing.heroTitle')}
          </h1>
          <p className="mb-8 max-w-xl text-base leading-relaxed text-[var(--sea-ink-soft)] sm:text-lg">
            {t('landing.heroLead')}
          </p>
          <div className="flex flex-col gap-3 sm:flex-row">
            <Button asChild size="lg" className="h-12 min-w-44 text-sm">
              <Link to="/login">{t('landing.primaryCta')}</Link>
            </Button>
            <Button asChild variant="outline" size="lg" className="h-12 min-w-44 text-sm">
              <Link to="/about">{t('landing.secondaryCta')}</Link>
            </Button>
          </div>
          <ul className="mt-10 grid gap-x-8 gap-y-3 border-t border-[var(--line)] pt-6">
            {FACTS.map((fact) => (
              <li key={fact} className="flex gap-3">
                <ArrowRight
                  className="mt-1 h-3.5 w-3.5 shrink-0 text-[var(--lagoon-deep)]"
                  aria-hidden="true"
                />
                <span className="text-sm leading-relaxed text-[var(--sea-ink-soft)]">
                  {t(fact)}
                </span>
              </li>
            ))}
          </ul>
        </div>
        <div className="rise-in lg:pt-6">
          <ModulePanel />
        </div>
      </div>
    </section>
  )
}
