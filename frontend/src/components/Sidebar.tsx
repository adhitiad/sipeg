import { Link, useLocation } from '@tanstack/react-router'
import {
  Briefcase,
  Building2,
  CalendarCheck,
  CalendarClock,
  CalendarDays,
  CalendarRange,
  FileText,
  Gauge,
  Heart,
  LayoutDashboard,
  GraduationCap,
  HeartPulse,
  LayoutGrid,
  Layers,
  Fingerprint,
  Receipt,
  ScrollText,
  Settings,
  Tags,
  UserCog,
  Users,
  Wallet,
  ClipboardList,
  FileSignature,
  Calculator,
  Gift,
  ShieldCheck,
  Palmtree,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  Sidebar as AppSidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
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

type NavGroup = {
  label: MessageKey
  items: NavItem[]
}

type SidebarProps = {
  account: { email: string; roleName?: string | null; permissions?: string[] }
}

/**
 * Navigasi aplikasi dikelompokkan menurut pekerjaan yang dilakukan, bukan
 * menurut modul backend. Setiap ikon dipilih karena dekat dengan isi
 * halamannya, bukan sebagai hiasan.
 */
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
  const canReadCuti =
    permissions.includes('cuti:read:any') ||
    permissions.includes('cuti:read:homebase') ||
    permissions.includes('cuti:read:self')
  const canReadAuditLogs =
    permissions.includes('audit:read') || permissions.includes('audit:read:homebase')
  const canManageRoles = permissions.includes('employee:role:manage')
  const canReadJabatan = permissions.includes('jabatan:read')
  const canManagePresensi = permissions.includes('attendance:presensi:manage')
  const canReadPemberhentian =
    permissions.includes('pemberhentian:read:any') ||
    permissions.includes('pemberhentian:read:homebase')
  const canManageKontrak = permissions.includes('perpanjangan:manage')
  const canManagePesangon = permissions.includes('pesangon:manage')
  const canManageThr = permissions.includes('thr:manage')
  const canManagePensiun =
    permissions.includes('pensiun:manage') ||
    permissions.includes('pensiun:read:homebase')
  const canReadSk =
    permissions.includes('sk:read:any') || permissions.includes('sk:read:homebase')
  const canReadRbac =
    permissions.includes('rbac:read') || permissions.includes('rbac:manage')
  const canReadKeluarga = permissions.some((permission: string) =>
    permission.startsWith('keluarga:read'),
  )
  const canReadArsip = permissions.some((permission: string) =>
    permission.startsWith('arsip:read'),
  )
  const canReadPayrollSlips = permissions.some((permission: string) =>
    permission.startsWith('payroll:read:'),
  )
  const canReadPayrollSchedules = permissions.some((permission: string) =>
    permission.startsWith('payroll:schedule'),
  )
  const canReadPayrollScale = permissions.some((permission: string) =>
    permission.startsWith('payroll:read:'),
  )
  const canReadSlo = permissions.includes('slo:read')

  const groups: NavGroup[] = [
    {
      label: 'nav.groupOverview',
      items: [
        {
          to: '/dashboard',
          key: 'nav.dashboard',
          icon: <LayoutDashboard aria-hidden="true" />,
        },
        {
          to: '/workspace',
          key: 'nav.workspace',
          icon: <LayoutGrid aria-hidden="true" />,
        },
      ],
    },
    {
      label: 'nav.groupPeople',
      items: [
        ...(canReadEmployees
          ? [
              {
                to: '/employees',
                key: 'nav.employees' as MessageKey,
                icon: <Users aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadJabatan
          ? [
              {
                to: '/jabatan',
                key: 'nav.jabatan' as MessageKey,
                icon: <Briefcase aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadOrganizations
          ? [
              {
                to: '/organizations',
                key: 'nav.organizations' as MessageKey,
                icon: <Building2 aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadCategories
          ? [
              {
                to: '/categories',
                key: 'nav.categories' as MessageKey,
                icon: <Tags aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadKeluarga
          ? [
              {
                to: '/keluarga',
                key: 'nav.keluarga' as MessageKey,
                icon: <Heart aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadArsip
          ? [
              {
                to: '/arsip',
                key: 'nav.arsip' as MessageKey,
                icon: <FileText aria-hidden="true" />,
              },
            ]
          : []),
      ],
    },
    {
      label: 'nav.groupActivities',
      items: [
        ...(canReadAttendance
          ? [
              {
                to: '/attendance/my-records',
                key: 'nav.attendance' as MessageKey,
                icon: <CalendarCheck aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadAttendanceSummary
          ? [
              {
                to: '/attendance/unit-summary',
                key: 'nav.attendanceSummary' as MessageKey,
                icon: <CalendarRange aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadCuti
          ? [
              {
                to: '/cuti',
                key: 'nav.cuti' as MessageKey,
                icon: <CalendarDays aria-hidden="true" />,
              },
            ]
          : []),
      ],
    },
    {
      label: 'nav.groupFinance',
      items: [
        ...(canReadPayrollSlips
          ? [
              {
                to: '/payroll/slips',
                key: 'nav.payrollSlips' as MessageKey,
                icon: <Wallet aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadPayrollSchedules
          ? [
              {
                to: '/payroll/schedules',
                key: 'nav.payrollSchedules' as MessageKey,
                icon: <CalendarClock aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadPayrollScale
          ? [
              {
                to: '/admin/payroll-scale',
                key: 'nav.payrollScale' as MessageKey,
                icon: <Layers aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadPayrollScale
          ? [
              {
                to: '/admin/honor-sks',
                key: 'nav.honorSks' as MessageKey,
                icon: <GraduationCap aria-hidden="true" />,
              },
              {
                to: '/admin/bpjs',
                key: 'nav.bpjs' as MessageKey,
                icon: <HeartPulse aria-hidden="true" />,
              },
              {
                to: '/admin/pph21-ter',
                key: 'nav.pph21Ter' as MessageKey,
                icon: <Receipt aria-hidden="true" />,
              },
            ]
          : []),
      ],
    },
    {
      label: 'nav.groupAdmin',
      items: [
        ...(canReadAuditLogs
          ? [
              {
                to: '/audit-logs',
                key: 'nav.auditLogs' as MessageKey,
                icon: <ScrollText aria-hidden="true" />,
              },
            ]
          : []),
        ...(canManageRoles
          ? [
              {
                to: '/admin/roles',
                key: 'nav.adminRoles' as MessageKey,
                icon: <UserCog aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadSlo
          ? [
              {
                to: '/admin/slo',
                key: 'nav.sloMetrics' as MessageKey,
                icon: <Gauge aria-hidden="true" />,
              },
            ]
          : []),
        ...(canManagePresensi
          ? [
              {
                to: '/admin/mesin-pin',
                key: 'nav.mesinPin' as MessageKey,
                icon: <Fingerprint aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadPemberhentian
          ? [
              {
                to: '/admin/pemberhentian',
                key: 'nav.pemberhentian' as MessageKey,
                icon: <ClipboardList aria-hidden="true" />,
              },
            ]
          : []),
        ...(canManageKontrak
          ? [
              {
                to: '/admin/kontrak',
                key: 'nav.kontrak' as MessageKey,
                icon: <FileSignature aria-hidden="true" />,
              },
            ]
          : []),
        ...(canManagePesangon
          ? [
              {
                to: '/admin/pesangon-aturan',
                key: 'nav.pesangonAturan' as MessageKey,
                icon: <Calculator aria-hidden="true" />,
              },
            ]
          : []),
        ...(canManageThr
          ? [
              {
                to: '/admin/thr',
                key: 'nav.thr' as MessageKey,
                icon: <Gift aria-hidden="true" />,
              },
            ]
          : []),
        ...(canManagePensiun
          ? [
              {
                to: '/admin/pensiun',
                key: 'nav.pensiun' as MessageKey,
                icon: <Palmtree aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadSk
          ? [
              {
                to: '/admin/sk',
                key: 'nav.sk' as MessageKey,
                icon: <ScrollText aria-hidden="true" />,
              },
            ]
          : []),
        ...(canReadRbac
          ? [
              {
                to: '/admin/rbac',
                key: 'nav.rbac' as MessageKey,
                icon: <ShieldCheck aria-hidden="true" />,
              },
            ]
          : []),
        {
          to: '/settings',
          key: 'nav.settings',
          icon: <Settings aria-hidden="true" />,
        },
      ],
    },
  ]

  const visibleGroups = groups.filter((group) => group.items.length > 0)

  return (
    <AppSidebar collapsible="icon">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild size="lg" tooltip={t('app.name' as MessageKey)}>
              <Link
                to="/"
                className="flex items-center gap-3 rounded-lg px-2 py-2 text-sm font-semibold tracking-tight text-[var(--sea-ink)] transition-colors"
              >
                <span
                  className="h-2 w-2 shrink-0 rounded-full bg-[var(--lagoon)]"
                  aria-hidden="true"
                />
                <span className="flex-1 truncate group-data-[collapsible=icon]:hidden">
                  {t('app.name' as MessageKey)}
                </span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        {visibleGroups.map((group) => (
          <SidebarGroup key={group.label}>
            <SidebarGroupLabel className="text-[0.68rem] tracking-[0.16em] text-[var(--sea-ink-soft)] uppercase">
              {t(group.label)}
            </SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {group.items.map((item) => (
                  <SidebarMenuItem key={item.to}>
                    <SidebarMenuButton
                      asChild
                      isActive={location.pathname === item.to}
                      tooltip={t(item.key)}
                    >
                      <Link
                        to={item.to}
                        className={cn(
                          'flex items-center gap-3 rounded-lg px-2 py-2 text-sm font-medium transition-colors',
                        )}
                      >
                        {item.icon}
                        <span className="flex-1 group-data-[collapsible=icon]:hidden">
                          {t(item.key)}
                        </span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
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
