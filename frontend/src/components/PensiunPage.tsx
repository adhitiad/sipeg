import { ShieldAlert } from 'lucide-react'
import { useMessages } from '@/i18n/locale-context'
import { useSession } from '@/lib/session'
import LoadingRadar from '@/components/LoadingRadar'
import PensiunKonfigurasiSection from '@/components/PensiunKonfigurasiSection'
import PensiunKepesertaanSection from '@/components/PensiunKepesertaanSection'

export default function PensiunPage() {
  const t = useMessages()
  const session = useSession()
  const permissions = session.data?.permissions ?? []
  const canWrite = permissions.includes('pensiun:manage')
  const canRead = canWrite || permissions.includes('pensiun:read:homebase')

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

  if (!session.data) {
    return (
      <div className="page-wrap flex flex-col items-center gap-4 px-4 py-10 sm:py-14">
        <LoadingRadar size="md" decorative />
      </div>
    )
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('pensiun.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('pensiun.title')}
      </h1>
      <p className="mb-6 max-w-3xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('pensiun.lead')}
      </p>
      <div className="grid gap-6 xl:grid-cols-2">
        <PensiunKonfigurasiSection canWrite={canWrite} />
        <PensiunKepesertaanSection canWrite={canWrite} />
      </div>
    </div>
  )
}
