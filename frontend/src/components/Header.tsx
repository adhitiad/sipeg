import { Link, useNavigate } from '@tanstack/react-router'
import { ArrowRight, LogOut } from 'lucide-react'
import { toast } from 'sonner'
import ThemeToggle from '@/components/ThemeToggle'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { SessionButton } from '@/components/ui/session-button'
import { SidebarTrigger } from '@/components/ui/sidebar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useLocale, useMessages } from '@/i18n/locale-context'
import { formatRoleName } from '@/lib/roles'
import { useSignOut, useSession } from '@/lib/session'
import type { Locale, MessageKey } from '@/i18n/dictionary'

const LOCALES: readonly Locale[] = ['id', 'en']

type HeaderProps = {
  account?: { email: string; roleName?: string | null; permissions?: string[] } | null
  withSidebarTrigger?: boolean
}

export default function Header({
  account: accountProp,
  withSidebarTrigger = false,
}: HeaderProps) {
  const t = useMessages()
  const { locale, setLocale } = useLocale()
  const session = useSession()
  const signOut = useSignOut()
  const navigate = useNavigate()

  const authAccount = session.data ?? accountProp ?? null
  const permissions: readonly string[] = authAccount?.permissions ?? []

  async function handleSignOut() {
    try {
      await signOut.mutateAsync()
      toast.success(t('login.signedOut'))
      await navigate({ to: '/' })
    } catch {
      toast.error(t('login.signOutFailed'))
    }
  }

  const canReadSlo = permissions.includes('slo:read')
  const canReadEmployees =
    permissions.some((permission: string) => permission.startsWith('employee:read:')) ??
    false
  const canReadAttendance = permissions.includes('attendance:read:self')
  const canReadAttendanceSummary =
    permissions.includes('attendance:read:any') ||
    permissions.includes('attendance:read:homebase')

  const email = authAccount?.email ?? ''

  return (
    <header className="sticky top-0 z-50 border-b border-[var(--line)] bg-[var(--header-bg)] px-4 backdrop-blur-lg">
      <nav
        aria-label={t('nav.primary')}
        className="page-wrap flex flex-wrap items-center gap-x-3 gap-y-2 py-3 sm:flex-nowrap sm:py-4"
      >
        {withSidebarTrigger ? (
          <SidebarTrigger aria-label={t('nav.openMenu')} className="-ml-1" />
        ) : null}
        <Link
          to="/"
          className="inline-flex shrink-0 items-center gap-2 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-2 text-sm font-semibold tracking-tight text-[var(--sea-ink)] no-underline sm:px-4"
        >
          <span className="h-2 w-2 rounded-full bg-[var(--lagoon)]" aria-hidden="true" />
          {t('app.name')}
        </Link>

        <div className="ml-auto flex items-center gap-2">
          <div
            role="group"
            aria-label={t('locale.switch')}
            className="inline-flex items-center rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)]"
          >
            {LOCALES.map((next) => (
              <button
                key={next}
                type="button"
                aria-pressed={locale === next}
                aria-label={next === 'id' ? t('locale.id') : t('locale.en')}
                onClick={() => setLocale(next)}
                className={
                  locale === next
                    ? 'h-11 w-11 rounded-full text-xs font-bold uppercase text-[var(--sea-ink)] shadow-[0_0_0_1px_var(--chip-line)_inset] sm:h-9 sm:w-9'
                    : 'h-11 w-11 rounded-full text-xs font-semibold uppercase text-[var(--sea-ink-soft)] transition hover:text-[var(--sea-ink)] sm:h-9 sm:w-9'
                }
              >
                {next}
              </button>
            ))}
          </div>

          <ThemeToggle />

          {authAccount ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SessionButton
                  variant="ghost"
                  size="sm"
                  className="h-11 gap-2 px-2 sm:h-9 w-auto"
                  aria-label={t('nav.account')}
                  label={
                    <>
                      <Avatar className="mr-2 h-8 w-8">
                        <AvatarFallback className="bg-[var(--lagoon-deep)] font-bold text-white">
                          {email.trim().charAt(0).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <span className="hidden max-w-[9rem] truncate text-sm font-medium md:inline">
                        {formatRoleName(authAccount.roleName ?? undefined, t)}
                      </span>
                    </>
                  }
                />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-64">
                <DropdownMenuLabel>
                  <span className="block truncate text-foreground">{email}</span>
                  <span className="block text-xs font-normal text-muted-foreground">
                    {formatRoleName(authAccount.roleName ?? undefined, t)}
                  </span>
                </DropdownMenuLabel>
                <DropdownMenuItem asChild>
                  <Link to="/account/mfa">{t('nav.accountSecurity' as MessageKey)}</Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <Link to="/workspace">{t('nav.workspace' as MessageKey)}</Link>
                </DropdownMenuItem>
                {canReadAttendance ? (
                  <DropdownMenuItem asChild>
                    <Link to="/attendance/my-records">
                      {t('nav.attendance' as MessageKey)}
                    </Link>
                  </DropdownMenuItem>
                ) : null}
                {canReadAttendanceSummary ? (
                  <DropdownMenuItem asChild>
                    <Link to="/attendance/unit-summary">
                      {t('nav.attendanceSummary' as MessageKey)}
                    </Link>
                  </DropdownMenuItem>
                ) : null}
                {canReadEmployees ? (
                  <DropdownMenuItem asChild>
                    <Link to="/employees">{t('nav.employees' as MessageKey)}</Link>
                  </DropdownMenuItem>
                ) : null}
                {canReadSlo ? (
                  <DropdownMenuItem asChild>
                    <Link to="/admin/slo">{t('nav.sloMetrics' as MessageKey)}</Link>
                  </DropdownMenuItem>
                ) : null}
                <DropdownMenuSeparator />
                <SessionButton
                  variant="ghost"
                  size="sm"
                  className="w-full justify-start"
                  onClick={handleSignOut}
                  label={t('nav.signOut' as MessageKey)}
                  icon={<LogOut className="h-4 w-4" aria-hidden="true" />}
                />
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <SessionButton
              className="h-11 sm:h-9 w-auto"
              onClick={() => navigate({ to: '/login' })}
              label={t('nav.login' as MessageKey)}
              icon={<ArrowRight className="w-5 h-5" aria-hidden="true" />}
            />
          )}
        </div>
      </nav>
    </header>
  )
}
