import { createFileRoute } from '@tanstack/react-router'
import { ShieldCheck } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { formatRoleName } from '@/lib/roles'
import { useSession } from '@/lib/session'
import { useMessages } from '@/i18n/locale-context'

export const Route = createFileRoute('/_app/workspace')({
  component: WorkspacePage,
})

function WorkspacePage() {
  const t = useMessages()
  const session = useSession()
  const account = session.data
  const permissions = account?.permissions ?? []

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <section className="island-shell rise-in rounded-2xl p-6 sm:p-8">
        <p className="island-kicker mb-3">{t('app.title')}</p>
        <h1 className="display-title mb-3 break-words text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
          {account ? t('app.signedInAs', { email: account.email }) : t('common.loading')}
        </h1>
        <dl className="mt-6 grid gap-4 sm:grid-cols-3">
          <div>
            <dt className="island-kicker mb-1">{t('app.emailLabel')}</dt>
            <dd className="m-0 break-all text-[var(--sea-ink)]">
              {account?.email ?? t('common.loading')}
            </dd>
          </div>
          <div>
            <dt className="island-kicker mb-1">{t('app.roleLabel')}</dt>
            <dd className="m-0 font-semibold text-[var(--sea-ink)]">
              {account?.roleName
                ? formatRoleName(account.roleName, t)
                : t('app.roleMissing')}
            </dd>
          </div>
          <div>
            <dt className="island-kicker mb-1">{t('app.unitLabel')}</dt>
            <dd className="m-0 break-all font-mono text-sm text-[var(--sea-ink)]">
              {account?.homebaseId ?? t('app.unitMissing')}
            </dd>
          </div>
        </dl>
      </section>

      <section className="mt-8">
        <Card className="island-shell border-[var(--line)]">
          <CardHeader>
            <CardDescription className="mb-1 flex items-center gap-2">
              <ShieldCheck className="h-4 w-4" aria-hidden="true" />
              {t('app.accessCount', { count: String(permissions.length) })}
            </CardDescription>
            <h2 className="text-lg leading-none font-semibold">{t('app.accessTitle')}</h2>
            <CardDescription>{t('app.accessLead')}</CardDescription>
          </CardHeader>
          <CardContent>
            {permissions.length === 0 ? (
              <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
                {t('app.accessEmpty')}
              </p>
            ) : (
              <ul className="m-0 flex list-none flex-wrap gap-2 p-0">
                {permissions.map((permission) => (
                  <li
                    key={permission}
                    className="rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-1.5 font-mono text-xs text-[var(--sea-ink)]"
                  >
                    {permission}
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </section>
    </div>
  )
}
