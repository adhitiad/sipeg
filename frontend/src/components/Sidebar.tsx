import { Link, useLocation } from '@tanstack/react-router'
import {
  BarChart3,
  Briefcase,
  CalendarClock,
  KeyRound,
  Receipt,
  ScrollText,
  ShieldAlert,
  Tags,
  User,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  Sidebar as AppSidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from '@/components/ui/sidebar'
import { useLocale, useMessages } from '@/i18n/locale-context'
import { useSession } from '@/lib/session'
import type { MessageKey } from '@/i18n/dictionary'

type NavItem = {
  to: string
  key: MessageKey
  icon: React.ReactNode
}

const navItems: NavItem[] = [
  { to: '/', key: 'nav.home', icon: <User aria-hidden="true" /> },
  { to: '/about', key: 'nav.about', icon: <ShieldAlert aria-hidden="true" /> },
  { to: '/workspace', key: 'nav.workspace', icon: <User aria-hidden="true" /> },
  {
    to: '/attendance/my-records',
    key: 'nav.attendance',
    icon: <ShieldAlert aria-hidden="true" />,
  },
  {
    to: '/attendance/unit-summary',
    key: 'nav.attendanceSummary',
    icon: <ShieldAlert aria-hidden="true" />,
  },
  { to: '/employees', key: 'nav.employees', icon: <User aria-hidden="true" /> },
  { to: '/admin/slo', key: 'nav.sloMetrics', icon: <ShieldAlert aria-hidden="true" /> },
]

const dashboardItem: NavItem = {
  to: '/dashboard',
  key: 'nav.dashboard',
  icon: <BarChart3 aria-hidden="true" />,
}
const workspaceItem: NavItem = {
  to: '/workspace',
  key: 'nav.workspace',
  icon: <User aria-hidden="true" />,
}
const attendanceItem: NavItem = {
  to: '/attendance/my-records',
  key: 'nav.attendance',
  icon: <ShieldAlert aria-hidden="true" />,
}
const attendanceSummaryItem: NavItem = {
  to: '/attendance/unit-summary',
  key: 'nav.attendanceSummary',
  icon: <ShieldAlert aria-hidden="true" />,
}
const employeesItem: NavItem = {
  to: '/employees',
  key: 'nav.employees',
  icon: <User aria-hidden="true" />,
}
const organizationsItem: NavItem = {
  to: '/organizations',
  key: 'nav.organizations',
  icon: <ShieldAlert aria-hidden="true" />,
}
const categoriesItem: NavItem = {
  to: '/categories',
  key: 'nav.categories',
  icon: <Tags aria-hidden="true" />,
}
const settingsItem: NavItem = {
  to: '/settings',
  key: 'nav.settings',
  icon: <KeyRound aria-hidden="true" />,
}
const auditItem: NavItem = {
  to: '/audit-logs',
  key: 'nav.auditLogs',
  icon: <ScrollText aria-hidden="true" />,
}
const jabatanItem: NavItem = {
  to: '/jabatan',
  key: 'nav.jabatan',
  icon: <Briefcase aria-hidden="true" />,
}
const rolesItem: NavItem = {
  to: '/admin/roles',
  key: 'nav.adminRoles',
  icon: <KeyRound aria-hidden="true" />,
}
const payrollSlipsItem: NavItem = {
  to: '/payroll/slips',
  key: 'nav.payrollSlips',
  icon: <Receipt aria-hidden="true" />,
}
const payrollSchedulesItem: NavItem = {
  to: '/payroll/schedules',
  key: 'nav.payrollSchedules',
  icon: <CalendarClock aria-hidden="true" />,
}
const sloItem: NavItem = {
  to: '/admin/slo',
  key: 'nav.sloMetrics',
  icon: <ShieldAlert aria-hidden="true" />,
}

type SidebarProps = {
  account: { email: string; roleName?: string | null; permissions?: string[] }
}

export default function Sidebar({ account }: SidebarProps) {
  const t = useMessages()
  const { locale, setLocale } = useLocale()
  const session = useSession()
  const location = useLocation()

  const permissions = session.data?.permissions ?? account.permissions ?? []
  const canReadAttendance = permissions.includes('attendance:read:self')
  const canReadAttendanceSummary =
    permissions.includes('attendance:read:any') ||
    permissions.includes('attendance:read:homebase')
  const canReadEmployees = permissions.some((permission: string) =>
    permission.startsWith('employee:read:'),
  )
  const canReadOrganizations =
    permissions.includes('organization:read') ||
    permissions.includes('organization:read:homebase')
  const canReadCategories = permissions.includes('category:read')
  const canReadAuditLogs =
    permissions.includes('audit:read') || permissions.includes('audit:read:homebase')
  const canManageRoles = permissions.includes('employee:role:manage')
  const canReadJabatan = permissions.includes('jabatan:read')
  const canReadPayrollSlips = permissions.some((permission: string) =>
    permission.startsWith('payroll:read:'),
  )
  const canReadPayrollSchedules = permissions.some((permission: string) =>
    permission.startsWith('payroll:schedule'),
  )
  const canReadSlo = permissions.includes('slo:read')

  const items: NavItem[] = [
    ...navItems.filter((item) => item.to === '/' || item.to === '/about'),
    workspaceItem,
    dashboardItem,
    ...(canReadAttendance ? [attendanceItem] : []),
    ...(canReadAttendanceSummary ? [attendanceSummaryItem] : []),
    settingsItem,
    ...(canReadEmployees ? [employeesItem] : []),
    ...(canReadOrganizations ? [organizationsItem] : []),
    ...(canReadCategories ? [categoriesItem] : []),
    ...(canReadJabatan ? [jabatanItem] : []),
    ...(canReadPayrollSlips ? [payrollSlipsItem] : []),
    ...(canReadPayrollSchedules ? [payrollSchedulesItem] : []),
    ...(canReadAuditLogs ? [auditItem] : []),
    ...(canManageRoles ? [rolesItem] : []),
    ...(canReadSlo ? [sloItem] : []),
  ]

  return (
    <AppSidebar collapsible="icon">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild size="lg" tooltip={t('app.name' as MessageKey)}>
              <Link
                to="/"
                className={cn(
                  'flex items-center gap-3 rounded-lg px-2 py-2 text-sm font-semibold tracking-tight text-[var(--sea-ink)] transition-colors',
                )}
              >
                <span
                  className="h-2 w-2 shrink-0 rounded-full bg-[var(--lagoon)]"
                  aria-hidden="true"
                />
                <span className="flex-1 truncate">{t('app.name' as MessageKey)}</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarMenu>
            {items.map((item) => (
              <SidebarMenuItem key={item.to}>
                <SidebarMenuButton
                  asChild
                  isActive={location.pathname === item.to}
                  tooltip={t(item.key as MessageKey)}
                >
                  <Link
                    to={item.to}
                    className={cn(
                      'flex items-center gap-3 rounded-lg px-2 py-2 text-sm font-medium transition-colors',
                    )}
                  >
                    {item.icon}
                    <span className="flex-1">{t(item.key as MessageKey)}</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <div
          role="group"
          aria-label={t('locale.switch')}
          className="inline-flex items-center rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)]"
        >
          {(['id', 'en'] as const).map((next) => (
            <button
              key={next}
              type="button"
              aria-pressed={locale === next}
              aria-label={next === 'id' ? t('locale.id') : t('locale.en')}
              onClick={() => setLocale(next)}
              className={
                locale === next
                  ? 'h-9 w-10 rounded-full text-xs font-bold uppercase text-[var(--sea-ink)] shadow-[0_0_0_1px_var(--chip-line)_inset] sm:h-8'
                  : 'h-9 w-10 rounded-full text-xs font-semibold uppercase text-[var(--sea-ink-soft)] transition hover:text-[var(--sea-ink)] sm:h-8'
              }
            >
              {next}
            </button>
          ))}
        </div>
      </SidebarFooter>
      <SidebarRail />
    </AppSidebar>
  )
}
