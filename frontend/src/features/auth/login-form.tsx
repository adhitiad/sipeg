import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { toast } from 'sonner'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import LoadingBars from '@/components/LoadingBars'
import { Button } from '@/components/ui/button'
import { SessionButton } from '@/components/ui/session-button'
import { Card, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useMessages } from '@/i18n/locale-context'
import { isAuthRuntimeConfigured, signInWithGoogle, useSignIn } from '@/lib/session'

export function LoginForm() {
  const t = useMessages()
  const navigate = useNavigate()
  const signIn = useSignIn()
  const runtimeConfigured = isAuthRuntimeConfigured()

  const loginSchema = useMemo(
    () =>
      z.object({
        email: z.email(t('login.emailInvalid')),
        password: z.string().min(1, t('login.passwordRequired')),
      }),
    [t],
  )

  type LoginFormValues = z.infer<typeof loginSchema>

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
  })

  const onSubmit = async (values: LoginFormValues) => {
    try {
      const outcome = await signIn.mutateAsync(values)
      if (outcome === 'twoFactor') {
        await navigate({ to: '/mfa' })
        return
      }
      toast.success(t('login.success'))
      await navigate({ to: '/workspace' })
    } catch (error) {
      const message =
        error instanceof Error ? error.message : t('login.invalidCredentials')
      toast.error(message)
    }
  }

  const handleGoogle = async () => {
    try {
      await signInWithGoogle()
    } catch (error) {
      const message =
        error instanceof Error ? error.message : t('login.googleUnavailable')
      toast.error(message)
    }
  }

  const errorMessage = signIn.error instanceof Error ? signIn.error.message : null

  return (
    <div className="w-full max-w-md">
      <Link
        to="/"
        className="mb-6 inline-flex shrink-0 items-center gap-2 rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-2 text-sm font-semibold tracking-tight text-[var(--sea-ink)] no-underline sm:px-4"
      >
        <span className="h-2 w-2 rounded-full bg-[var(--lagoon)]" aria-hidden="true" />
        {t('app.name')}
      </Link>

      <Card className="island-shell border-[var(--line)]">
        <CardHeader>
          <h1 className="display-title text-2xl leading-none font-semibold text-[var(--sea-ink)]">
            {t('login.title')}
          </h1>
          <CardDescription>{t('login.lead')}</CardDescription>
        </CardHeader>
        <CardContent>
          {!runtimeConfigured ? (
            <Alert className="mb-5">
              <AlertTitle>{t('login.runtimeUnavailable')}</AlertTitle>
            </Alert>
          ) : null}

          {errorMessage ? (
            <Alert variant="destructive" className="mb-5">
              <AlertDescription>{errorMessage}</AlertDescription>
            </Alert>
          ) : null}

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-5" noValidate>
            <div className="space-y-2">
              <Label htmlFor="email">{t('login.email')}</Label>
              <Input
                id="email"
                type="email"
                autoComplete="email"
                placeholder={t('login.emailPlaceholder')}
                disabled={isSubmitting || !runtimeConfigured}
                aria-invalid={errors.email ? true : undefined}
                aria-describedby={errors.email ? 'email-error' : undefined}
                {...register('email')}
              />
              {errors.email ? (
                <p id="email-error" className="text-sm text-destructive">
                  {errors.email.message}
                </p>
              ) : null}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">{t('login.password')}</Label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                disabled={isSubmitting || !runtimeConfigured}
                aria-invalid={errors.password ? true : undefined}
                aria-describedby={errors.password ? 'password-error' : undefined}
                {...register('password')}
              />
              {errors.password ? (
                <p id="password-error" className="text-sm text-destructive">
                  {errors.password.message}
                </p>
              ) : null}
            </div>

            <SessionButton
              type="submit"
              className="h-11 w-full sm:h-9"
              disabled={isSubmitting || signIn.isPending || !runtimeConfigured}
              label={
                isSubmitting || signIn.isPending ? (
                  <>
                    <LoadingBars size="sm" decorative className="mr-2" />
                    {t('login.submitting')}
                  </>
                ) : (
                  t('login.submit')
                )
              }
              icon={<ArrowRight className="w-5 h-5" aria-hidden="true" />}
            />
          </form>

          <div className="my-4 flex items-center gap-3" aria-hidden="true">
            <span className="h-px flex-1 bg-[var(--line)]" />
            <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--sea-ink-soft)]">
              {t('login.or')}
            </span>
            <span className="h-px flex-1 bg-[var(--line)]" />
          </div>
          <Button
            type="button"
            variant="outline"
            className="h-11 w-full sm:h-9"
            onClick={() => void handleGoogle()}
            disabled={!runtimeConfigured}
          >
            {t('login.google')}
          </Button>

          <p className="mt-6 text-sm text-muted-foreground">{t('login.support')}</p>
        </CardContent>
      </Card>
    </div>
  )
}
