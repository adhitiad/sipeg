import { useState } from 'react'
import { createFileRoute, useNavigate, Link } from '@tanstack/react-router'
import { ShieldCheck } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useMessages } from '@/i18n/locale-context'
import { useQueryClient } from '@tanstack/react-query'
import { verifyBackupCode, verifyTotp, sessionQueryKey } from '@/lib/session'

export const Route = createFileRoute('/_auth/mfa')({
  component: MfaChallengePage,
})

function MfaChallengePage() {
  const t = useMessages()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [mode, setMode] = useState<'totp' | 'backup'>('totp')
  const [code, setCode] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const trimmed = code.trim().replace(/\s/g, '')
    if (!trimmed) {
      return
    }
    setIsSubmitting(true)
    try {
      if (mode === 'totp') {
        await verifyTotp(trimmed)
      } else {
        await verifyBackupCode(trimmed)
      }
      await queryClient.invalidateQueries({ queryKey: sessionQueryKey })
      toast.success(t('login.success'))
      await navigate({ to: '/workspace' })
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('mfa.verifyError'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="island-shell w-full max-w-md rounded-2xl p-6 sm:p-8">
      <div className="mb-6 flex flex-col items-center gap-3 text-center">
        <span className="flex h-14 w-14 items-center justify-center rounded-full bg-[var(--palm)] text-[var(--sidebar-primary-foreground)]">
          <ShieldCheck className="h-7 w-7" aria-hidden="true" />
        </span>
        <p className="island-kicker">{t('mfa.kicker')}</p>
        <h1 className="display-title text-2xl font-semibold tracking-tight text-[var(--sea-ink)]">
          {t('mfa.title')}
        </h1>
        <p className="m-0 max-w-sm text-sm leading-relaxed text-[var(--sea-ink-soft)]">
          {mode === 'totp' ? t('mfa.description') : t('mfa.backupDescription')}
        </p>
      </div>

      <form onSubmit={handleSubmit} className="grid gap-4" noValidate>
        <div>
          <Label htmlFor="mfa-code">{t('mfa.codeLabel')}</Label>
          <Input
            id="mfa-code"
            type="text"
            inputMode="numeric"
            autoComplete="one-time-code"
            autoFocus
            value={code}
            onChange={(event) => setCode(event.target.value)}
            placeholder={
              mode === 'totp' ? t('mfa.codePlaceholder') : t('mfa.backupPlaceholder')
            }
            className="h-12 text-center text-lg tracking-[0.35em]"
            maxLength={mode === 'totp' ? 6 : 24}
          />
        </div>
        <Button
          type="submit"
          className="h-11"
          disabled={isSubmitting || code.trim().length === 0}
        >
          {t('mfa.verify')}
        </Button>
      </form>

      <div className="mt-6 flex flex-col items-center gap-2 text-sm">
        <button
          type="button"
          className="text-[var(--lagoon-deep)] underline-offset-2 hover:underline"
          onClick={() => {
            setMode((current) => (current === 'totp' ? 'backup' : 'totp'))
            setCode('')
          }}
        >
          {mode === 'totp' ? t('mfa.useBackupCode') : t('mfa.useTotp')}
        </button>
        <Link
          to="/login"
          className="text-[var(--sea-ink-soft)] underline-offset-2 hover:underline"
        >
          {t('mfa.backToLogin')}
        </Link>
      </div>
    </div>
  )
}
