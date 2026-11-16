import { createFileRoute } from '@tanstack/react-router'
import { socialMeta } from '@/lib/seo'
import { DEFAULT_LOCALE, translate, type MessageKey } from '@/i18n/dictionary'
import { LoginForm } from '@/features/auth/login-form'

const t = (key: MessageKey) => translate(DEFAULT_LOCALE, key)

export const Route = createFileRoute('/_auth/login')({
  component: LoginPage,
  head: () => ({
    meta: [
      { title: t('seo.loginTitle') },
      { name: 'description', content: t('seo.loginDescription') },
      ...socialMeta({
        title: t('seo.loginTitle'),
        description: t('seo.loginDescription'),
        path: '/login',
      }),
    ],
  }),
})

function LoginPage() {
  return <LoginForm />
}
