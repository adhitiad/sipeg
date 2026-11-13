import { createFileRoute } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import ProfileForm from '@/components/ProfileForm'
import ChangePasswordForm from '@/components/ChangePasswordForm'
import { useThemeMode, setThemeMode } from '@/lib/theme'
import { useSession } from '@/lib/session'
import { useLocale, useMessages } from '@/i18n/locale-context'

export const Route = createFileRoute('/_app/settings')({
  component: SettingsPage,
})

function SettingsPage() {
  const t = useMessages()
  const { locale, setLocale } = useLocale()
  const mode = useThemeMode()
  const session = useSession()
  const account = session.data ?? null

  const isDark = mode === 'dark'

  function handleThemeChange(value: 'light' | 'dark') {
    setThemeMode(value)
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <Tabs defaultValue="account" className="w-full max-w-3xl">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="appearance">{t('settings.appearanceTitle')}</TabsTrigger>
          <TabsTrigger value="language">{t('settings.languageTitle')}</TabsTrigger>
          <TabsTrigger value="account">{t('settings.accountTitle')}</TabsTrigger>
          <TabsTrigger value="profile">{t('settings.profileTitle')}</TabsTrigger>
          <TabsTrigger value="security">{t('settings.securityTitle')}</TabsTrigger>
        </TabsList>

        <TabsContent value="appearance" className="space-y-3 mt-6">
          <Card className="island-shell">
            <CardHeader>
              <CardTitle>{t('settings.appearanceTitle')}</CardTitle>
              <CardDescription>{t('settings.appearanceDescription')}</CardDescription>
            </CardHeader>
            <CardContent className="grid items-center gap-3 py-4">
              <Switch
                checked={isDark}
                onCheckedChange={(checked) =>
                  handleThemeChange(checked ? 'dark' : 'light')
                }
                aria-checked={isDark}
              />
              <span className="text-sm font-medium text-[var(--sea-ink)]">
                {isDark ? t('theme.dark') : t('theme.light')}
              </span>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="language" className="space-y-3 mt-6">
          <Card className="island-shell">
            <CardHeader>
              <CardTitle>{t('settings.languageTitle')}</CardTitle>
              <CardDescription>{t('settings.languageDescription')}</CardDescription>
            </CardHeader>
            <CardContent className="grid items-center gap-3 py-4">
              <Select
                value={locale}
                onValueChange={setLocale}
                aria-label={t('settings.languageTitle')}
              >
                <SelectTrigger className="w-48 h-11">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="id">{t('locale.id')}</SelectItem>
                  <SelectItem value="en">{t('locale.en')}</SelectItem>
                </SelectContent>
              </Select>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="account" className="space-y-3 mt-6">
          <Card className="island-shell">
            <CardHeader>
              <CardTitle>{t('settings.accountTitle')}</CardTitle>
              <CardDescription>{t('settings.accountDescription')}</CardDescription>
            </CardHeader>
            <CardContent className="py-4 space-y-4">
              <dl className="grid gap-4 sm:grid-cols-2">
                <div>
                  <dt className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
                    {t('app.emailLabel')}
                  </dt>
                  <dd className="text-sm text-[var(--sea-ink)] break-all">
                    {account?.email ?? t('common.loading')}
                  </dd>
                </div>
                <div>
                  <dt className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
                    {t('app.roleLabel')}
                  </dt>
                  <dd className="text-sm font-medium text-[var(--sea-ink)]">
                    {account?.roleName ?? t('app.roleMissing')}
                  </dd>
                </div>
                <div>
                  <dt className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
                    {t('app.unitLabel')}
                  </dt>
                  <dd className="text-sm font-mono text-[var(--sea-ink)] break-all">
                    {account?.homebaseId ?? t('app.unitMissing')}
                  </dd>
                </div>
              </dl>
              <p className="m-0 text-sm leading-relaxed text-[var(--sea-ink-soft)]">
                {t('settings.accountDescription')}
              </p>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="profile" className="space-y-3 mt-6">
          <Card className="island-shell">
            <CardHeader>
              <CardTitle>{t('settings.profileTitle')}</CardTitle>
              <CardDescription>{t('settings.profileDescription')}</CardDescription>
            </CardHeader>
            <CardContent className="py-4">
              <ProfileForm />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="space-y-3 mt-6">
          <Card className="island-shell">
            <CardHeader>
              <CardTitle>{t('settings.securityTitle')}</CardTitle>
              <CardDescription>{t('settings.securityDescription')}</CardDescription>
            </CardHeader>
            <CardContent className="py-4 space-y-4">
              <ChangePasswordForm />
              <div className="pt-2">
                <Button asChild variant="outline" size="sm">
                  <a
                    href="/_app/account/mfa"
                    className="block text-sm font-medium text-[var(--sea-ink)] underline decoration-[1px] decoration-[var(--line)] hover:text-[var(--lagoon)]"
                  >
                    {t('settings.mfaButton')}
                  </a>
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <p className="mt-8 text-xs text-[var(--sea-ink-soft)]">
        {t('settings.mfaDescription')}
      </p>
    </div>
  )
}
