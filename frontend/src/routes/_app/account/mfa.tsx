import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { ShieldCheck, ShieldOff, Copy, Check } from 'lucide-react'
import { toast } from 'sonner'
import QRCode from 'qrcode'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useMessages } from '@/i18n/locale-context'
import {
  disableTwoFactor,
  enableTwoFactor,
  sessionQueryKey,
  verifyTotp,
  type EnableTwoFactorResponse,
} from '@/lib/session'
import { useQueryClient } from '@tanstack/react-query'

export const Route = createFileRoute('/_app/account/mfa')({
  component: AccountMfaPage,
})

type Phase = 'idle' | 'enrolling' | 'enrolled'

function AccountMfaPage() {
  const t = useMessages()
  const queryClient = useQueryClient()
  const [phase, setPhase] = useState<Phase>('idle')
  const [password, setPassword] = useState('')
  const [enrollment, setEnrollment] = useState<EnableTwoFactorResponse | null>(null)
  const [qrDataUrl, setQrDataUrl] = useState<string | null>(null)
  const [code, setCode] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [codesCopied, setCodesCopied] = useState(false)

  useEffect(() => {
    let cancelled = false
    if (enrollment) {
      QRCode.toDataURL(enrollment.totpURI, { width: 200, margin: 1 }).then((url) => {
        if (!cancelled) {
          setQrDataUrl(url)
        }
      })
    }
    return () => {
      cancelled = true
    }
  }, [enrollment])

  async function handleEnable() {
    setIsSubmitting(true)
    try {
      const result = await enableTwoFactor(password)
      setEnrollment(result)
      setPhase('enrolling')
      setPassword('')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('accountMfa.genericError'))
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleVerifyEnrollment(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const trimmed = code.trim()
    if (!trimmed) {
      return
    }
    setIsSubmitting(true)
    try {
      await verifyTotp(trimmed)
      await queryClient.invalidateQueries({ queryKey: sessionQueryKey })
      setPhase('enrolled')
      setEnrollment(null)
      setCode('')
      toast.success(t('accountMfa.enableSuccess'))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('accountMfa.genericError'))
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleDisable() {
    setIsSubmitting(true)
    try {
      await disableTwoFactor(password)
      setPassword('')
      setPhase('idle')
      toast.success(t('accountMfa.disableSuccess'))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('accountMfa.genericError'))
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleCopyCodes() {
    if (!enrollment?.backupCodes?.length) {
      return
    }
    try {
      await navigator.clipboard.writeText(enrollment.backupCodes.join('\n'))
      setCodesCopied(true)
      setTimeout(() => setCodesCopied(false), 2000)
    } catch {
      toast.error(t('accountMfa.genericError'))
    }
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('accountMfa.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('accountMfa.title')}
      </h1>
      <p className="mb-6 max-w-2xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('accountMfa.lead')}
      </p>

      {phase === 'enrolling' && enrollment ? (
        <Card className="max-w-lg">
          <CardHeader>
            <CardTitle>{t('accountMfa.verifyTitle')}</CardTitle>
            <CardDescription>{t('accountMfa.verifyDescription')}</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4">
            {qrDataUrl ? (
              <img
                src={qrDataUrl}
                alt={t('accountMfa.qrAlt')}
                width={200}
                height={200}
                className="mx-auto rounded-xl border border-[var(--line)]"
              />
            ) : null}
            <div>
              <Label htmlFor="totp-uri">{t('accountMfa.uriLabel')}</Label>
              <Input
                id="totp-uri"
                readOnly
                value={enrollment.totpURI}
                className="h-11 font-mono text-xs"
              />
            </div>
            {enrollment.backupCodes?.length ? (
              <div>
                <div className="mb-2 flex items-center justify-between gap-3">
                  <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
                    {t('accountMfa.backupCodesTitle')}
                  </span>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={handleCopyCodes}
                  >
                    {codesCopied ? (
                      <Check className="h-4 w-4" aria-hidden="true" />
                    ) : (
                      <Copy className="h-4 w-4" aria-hidden="true" />
                    )}
                    {t(codesCopied ? 'accountMfa.copied' : 'accountMfa.copyCodes')}
                  </Button>
                </div>
                <p className="mb-2 text-xs leading-relaxed text-[var(--sea-ink-soft)]">
                  {t('accountMfa.backupCodesNote')}
                </p>
                <div className="grid grid-cols-2 gap-1 rounded-xl border border-[var(--line)] p-3 font-mono text-sm sm:grid-cols-5">
                  {enrollment.backupCodes.map((codeValue: string) => (
                    <span key={codeValue}>{codeValue}</span>
                  ))}
                </div>
              </div>
            ) : null}
            <form onSubmit={handleVerifyEnrollment} className="grid gap-3" noValidate>
              <div>
                <Label htmlFor="verify-code">{t('accountMfa.totpCodeLabel')}</Label>
                <Input
                  id="verify-code"
                  type="text"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  value={code}
                  onChange={(event) => setCode(event.target.value)}
                  className="h-11 tracking-[0.35em]"
                  placeholder="000000"
                />
              </div>
              <Button type="submit" disabled={isSubmitting || !code.trim()}>
                {t('accountMfa.verifyButton')}
              </Button>
            </form>
          </CardContent>
        </Card>
      ) : phase === 'enrolled' ? (
        <Card className="max-w-lg">
          <CardHeader>
            <div className="flex items-center gap-3">
              <ShieldCheck className="h-6 w-6 text-[var(--palm)]" aria-hidden="true" />
              <CardTitle>{t('accountMfa.enrolledTitle')}</CardTitle>
            </div>
            <CardDescription>{t('accountMfa.enrolledDescription')}</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3">
            <div>
              <Label htmlFor="disable-password">{t('accountMfa.passwordLabel')}</Label>
              <Input
                id="disable-password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                className="h-11"
              />
            </div>
            <Button
              type="button"
              variant="destructive"
              disabled={isSubmitting || !password}
              onClick={() => void handleDisable()}
            >
              <ShieldOff className="h-4 w-4" aria-hidden="true" />
              {t('accountMfa.disableButton')}
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Card className="max-w-lg">
          <CardHeader>
            <CardTitle>{t('accountMfa.enableTitle')}</CardTitle>
            <CardDescription>{t('accountMfa.enableDescription')}</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3">
            <div>
              <Label htmlFor="enable-password">{t('accountMfa.passwordLabel')}</Label>
              <Input
                id="enable-password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                className="h-11"
              />
            </div>
            <Button
              type="button"
              disabled={isSubmitting || !password}
              onClick={() => void handleEnable()}
            >
              <ShieldCheck className="h-4 w-4" aria-hidden="true" />
              {t('accountMfa.enableButton')}
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
