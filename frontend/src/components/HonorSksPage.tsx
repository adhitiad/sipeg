import { ShieldAlert } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useMessages } from '@/i18n/locale-context'
import HonorPeriodeSection from '@/components/HonorPeriodeSection'
import HonorRateSection from '@/components/HonorRateSection'
import { useSession } from '@/lib/session'

export default function HonorSksPage() {
  const t = useMessages()
  const session = useSession()
  const permissions = session.data?.permissions ?? []
  const canRead =
    permissions.includes('payroll:read:any') ||
    permissions.includes('payroll:read:homebase') ||
    permissions.includes('payroll:manage')
  const canManage = permissions.includes('payroll:manage')

  if (session.data && !canRead) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <ShieldAlert
            className="mx-auto mb-3 h-8 w-8 text-[var(--sea-ink-soft)]"
            aria-hidden="true"
          />
          <h1 className="display-title mb-2 text-2xl font-bold text-[var(--sea-ink)]">
            {t('state.noAccess.title')}
          </h1>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('state.noAccess.description')}
          </p>
        </section>
      </div>
    )
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('honorSks.kicker')}</p>
      <h1 className="display-title mb-2 text-3xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('honorSks.title')}
      </h1>
      <p className="m-0 mb-6 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)]">
        {t('honorSks.lead')}
      </p>

      <Tabs defaultValue="rates" className="island-shell rounded-2xl p-4 sm:p-6">
        <TabsList>
          <TabsTrigger value="rates">{t('honorSks.tabRates')}</TabsTrigger>
          <TabsTrigger value="periode">{t('honorSks.tabPeriode')}</TabsTrigger>
        </TabsList>
        <TabsContent value="rates">
          <HonorRateSection canManage={canManage} />
        </TabsContent>
        <TabsContent value="periode">
          <HonorPeriodeSection canManage={canManage} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
