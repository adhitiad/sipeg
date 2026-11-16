import {
  Building2,
  CalendarCheck,
  CalendarDays,
  ClipboardCheck,
  FolderOpen,
  ScrollText,
  Users,
  Wallet,
} from 'lucide-react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'

/**
 * Ringkasan kelompok data yang dipegang SIMPEG. Ikon memakai satu pustaka
 * (lucide) supaya bentuknya seragam, dan tiap ikon dipilih karena maknanya
 * memang dekat dengan domainnya.
 */
const MODULES: readonly {
  title: MessageKey
  body: MessageKey
  icon: typeof Users
}[] = [
  {
    title: 'landing.scope.employee.title',
    body: 'landing.scope.employee.body',
    icon: Users,
  },
  {
    title: 'landing.scope.organization.title',
    body: 'landing.scope.organization.body',
    icon: Building2,
  },
  {
    title: 'landing.scope.attendance.title',
    body: 'landing.scope.attendance.body',
    icon: CalendarCheck,
  },
  {
    title: 'landing.scope.leave.title',
    body: 'landing.scope.leave.body',
    icon: CalendarDays,
  },
  {
    title: 'landing.scope.payroll.title',
    body: 'landing.scope.payroll.body',
    icon: Wallet,
  },
  {
    title: 'landing.scope.performance.title',
    body: 'landing.scope.performance.body',
    icon: ClipboardCheck,
  },
  {
    title: 'landing.scope.archive.title',
    body: 'landing.scope.archive.body',
    icon: FolderOpen,
  },
  {
    title: 'landing.scope.audit.title',
    body: 'landing.scope.audit.body',
    icon: ScrollText,
  },
]

export default function ModulePanel() {
  const t = useMessages()

  return (
    <Card className="border-[var(--line)] bg-[var(--surface-strong)] shadow-none">
      <CardHeader>
        <CardTitle className="display-title text-lg font-semibold text-[var(--sea-ink)]">
          {t('landing.scopeTitle')}
        </CardTitle>
        <CardDescription className="text-sm leading-relaxed text-[var(--sea-ink-soft)]">
          {t('landing.scopeLead')}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ul className="grid gap-x-6 gap-y-4 sm:grid-cols-2">
          {MODULES.map((module) => {
            const Icon = module.icon
            return (
              <li key={module.title} className="flex gap-3">
                <span
                  className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-[var(--accent)] text-[var(--lagoon-deep)]"
                  aria-hidden="true"
                >
                  <Icon className="h-4 w-4" />
                </span>
                <span className="min-w-0">
                  <span className="block text-sm font-semibold text-[var(--sea-ink)]">
                    {t(module.title)}
                  </span>
                  <span className="mt-0.5 block text-xs leading-relaxed text-[var(--sea-ink-soft)]">
                    {t(module.body)}
                  </span>
                </span>
              </li>
            )
          })}
        </ul>
      </CardContent>
    </Card>
  )
}
