import { Building2, ShieldCheck, User, UserCheck } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'

/**
 * Peran beserta cakupan datanya. Disusun sebagai daftar bertatar, bukan
 * kartu seragam: nama peran dan cakupan dibaca lebih dulu, penjelasan
 * menyusul. Baris administrator diberi latar karena cakupannya berbeda
 * jenis, bukan sekadar lebih besar.
 */
const ROLES: readonly {
  title: MessageKey
  body: MessageKey
  scope: MessageKey
  icon: typeof User
  highlight?: boolean
}[] = [
  {
    title: 'landing.role.admin.title',
    body: 'landing.role.admin.body',
    scope: 'landing.role.admin.scope',
    icon: ShieldCheck,
    highlight: true,
  },
  {
    title: 'landing.role.operator.title',
    body: 'landing.role.operator.body',
    scope: 'landing.role.operator.scope',
    icon: Building2,
  },
  {
    title: 'landing.role.atasan.title',
    body: 'landing.role.atasan.body',
    scope: 'landing.role.atasan.scope',
    icon: UserCheck,
  },
  {
    title: 'landing.role.pegawai.title',
    body: 'landing.role.pegawai.body',
    scope: 'landing.role.pegawai.scope',
    icon: User,
  },
]

export default function RolesSection() {
  const t = useMessages()

  return (
    <section className="page-wrap px-4 pb-16 sm:pb-24">
      <h2 className="display-title mb-3 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-3xl">
        {t('landing.rolesTitle')}
      </h2>
      <p className="mb-8 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)] sm:text-base">
        {t('landing.rolesLead')}
      </p>
      <dl className="overflow-hidden rounded-2xl border border-[var(--line)]">
        {ROLES.map((role) => {
          const Icon = role.icon
          return (
            <div
              key={role.title}
              className={
                role.highlight
                  ? 'grid gap-3 border-b border-[var(--line)] bg-[var(--chip-bg)] px-5 py-5 sm:grid-cols-[minmax(0,16rem)_minmax(0,1fr)] sm:gap-8'
                  : 'grid gap-3 border-b border-[var(--line)] px-5 py-5 last:border-b-0 sm:grid-cols-[minmax(0,16rem)_minmax(0,1fr)] sm:gap-8'
              }
            >
              <dt className="flex items-start gap-3">
                <span
                  className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-[var(--accent)] text-[var(--lagoon-deep)]"
                  aria-hidden="true"
                >
                  <Icon className="h-4.5 w-4.5" />
                </span>
                <span>
                  <span className="block text-base font-semibold text-[var(--sea-ink)]">
                    {t(role.title)}
                  </span>
                  <Badge
                    variant="secondary"
                    className="mt-1.5 border-[var(--chip-line)] bg-[var(--surface-strong)] text-[var(--sea-ink-soft)]"
                  >
                    {t(role.scope)}
                  </Badge>
                </span>
              </dt>
              <dd className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)] sm:pt-1">
                {t(role.body)}
              </dd>
            </div>
          )
        })}
      </dl>
    </section>
  )
}
