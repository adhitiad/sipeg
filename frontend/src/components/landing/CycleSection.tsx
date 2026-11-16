import { CalendarClock, ClipboardCheck, Users, Wallet } from 'lucide-react'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'

/**
 * Urutan kerja nyata kantor kepegawaian. Ikon sengaja memakai simbol yang
 * sama dengan panel modul agar dua bagian terbaca sebagai satu sistem.
 */
const STEPS: readonly {
  title: MessageKey
  body: MessageKey
  icon: typeof Users
}[] = [
  {
    title: 'landing.cycle.data.title',
    body: 'landing.cycle.data.body',
    icon: Users,
  },
  {
    title: 'landing.cycle.daily.title',
    body: 'landing.cycle.daily.body',
    icon: CalendarClock,
  },
  {
    title: 'landing.cycle.payroll.title',
    body: 'landing.cycle.payroll.body',
    icon: Wallet,
  },
  {
    title: 'landing.cycle.evaluation.title',
    body: 'landing.cycle.evaluation.body',
    icon: ClipboardCheck,
  },
]

export default function CycleSection() {
  const t = useMessages()

  return (
    <section className="page-wrap px-4 pb-16 sm:pb-24">
      <div className="grid gap-10 lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1fr)] lg:gap-16">
        <div>
          <h2 className="display-title mb-3 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-3xl">
            {t('landing.cycleTitle')}
          </h2>
          <p className="mb-4 max-w-xl text-sm leading-relaxed text-[var(--sea-ink-soft)] sm:text-base">
            {t('landing.cycleLead')}
          </p>
          <p className="m-0 max-w-xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
            {t('landing.cycleIntro')}
          </p>
        </div>
        <ol className="relative space-y-6 border-l border-[var(--line)] pl-6">
          {STEPS.map((step) => {
            const Icon = step.icon
            return (
              <li key={step.title} className="relative">
                <span
                  className="absolute top-1 -left-[1.9rem] flex h-8 w-8 items-center justify-center rounded-full border border-[var(--line)] bg-[var(--surface-strong)] text-[var(--lagoon-deep)]"
                  aria-hidden="true"
                >
                  <Icon className="h-4 w-4" />
                </span>
                <h3 className="mb-1 text-base font-semibold text-[var(--sea-ink)]">
                  {t(step.title)}
                </h3>
                <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
                  {t(step.body)}
                </p>
              </li>
            )
          })}
        </ol>
      </div>
    </section>
  )
}
