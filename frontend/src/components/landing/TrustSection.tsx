import { KeyRound, ScrollText, ShieldCheck } from 'lucide-react'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'

/** Tiga hal yang membuat akses dan perubahan dapat ditelusuri. */
const POINTS: readonly {
  title: MessageKey
  body: MessageKey
  icon: typeof KeyRound
}[] = [
  {
    title: 'landing.access.account.title',
    body: 'landing.access.account.body',
    icon: KeyRound,
  },
  {
    title: 'landing.access.role.title',
    body: 'landing.access.role.body',
    icon: ShieldCheck,
  },
  {
    title: 'landing.access.audit.title',
    body: 'landing.access.audit.body',
    icon: ScrollText,
  },
]

export default function TrustSection() {
  const t = useMessages()

  return (
    <section className="page-wrap px-4 pb-16 sm:pb-24">
      <div className="rounded-2xl border border-[var(--line)] bg-[var(--surface)] px-6 py-8 sm:px-10 sm:py-10">
        <h2 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)]">
          {t('landing.accessTitle')}
        </h2>
        <p className="mb-8 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
          {t('landing.accessLead')}
        </p>
        <div className="grid gap-8 sm:grid-cols-3">
          {POINTS.map((point) => {
            const Icon = point.icon
            return (
              <div key={point.title}>
                <Icon
                  className="mb-3 h-5 w-5 text-[var(--lagoon-deep)]"
                  aria-hidden="true"
                />
                <h3 className="mb-1.5 text-sm font-semibold text-[var(--sea-ink)]">
                  {t(point.title)}
                </h3>
                <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
                  {t(point.body)}
                </p>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}
