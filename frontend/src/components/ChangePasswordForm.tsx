import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { KeyRound } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'
import { AuthRuntimeError, changePassword } from '@/lib/session'
import { changePasswordSchema, type ChangePasswordInput } from '@/validations/schemas'

export default function ChangePasswordForm() {
  const t = useMessages()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const form = useForm<ChangePasswordInput>({
    resolver: zodResolver(changePasswordSchema),
    defaultValues: {
      currentPassword: '',
      newPassword: '',
      confirmPassword: '',
    },
  })
  const { errors } = form.formState
  const message = (key: string | undefined) => (key ? t(key as MessageKey) : null)

  async function onSubmit(values: ChangePasswordInput) {
    setIsSubmitting(true)
    try {
      await changePassword(values)
      form.reset()
      toast.success(t('settings.changePasswordSuccess'))
    } catch (error) {
      toast.error(
        error instanceof AuthRuntimeError && error.message.length > 0
          ? error.message
          : t('settings.changePasswordError'),
      )
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form
      className="grid gap-4 md:grid-cols-3"
      noValidate
      onSubmit={form.handleSubmit(onSubmit)}
    >
      <div className="grid gap-1.5">
        <Label htmlFor="settings-current-password">
          {t('settings.currentPasswordLabel')}
        </Label>
        <Input
          id="settings-current-password"
          type="password"
          autoComplete="current-password"
          aria-invalid={errors.currentPassword ? true : undefined}
          aria-describedby={
            errors.currentPassword ? 'settings-current-password-error' : undefined
          }
          className="h-11"
          {...form.register('currentPassword')}
        />
        {errors.currentPassword ? (
          <p id="settings-current-password-error" className="m-0 text-xs text-red-600">
            {message(errors.currentPassword.message)}
          </p>
        ) : null}
      </div>

      <div className="grid gap-1.5">
        <Label htmlFor="settings-new-password">{t('settings.newPasswordLabel')}</Label>
        <Input
          id="settings-new-password"
          type="password"
          autoComplete="new-password"
          aria-invalid={errors.newPassword ? true : undefined}
          aria-describedby={
            errors.newPassword ? 'settings-new-password-error' : undefined
          }
          className="h-11"
          {...form.register('newPassword')}
        />
        {errors.newPassword ? (
          <p id="settings-new-password-error" className="m-0 text-xs text-red-600">
            {message(errors.newPassword.message)}
          </p>
        ) : null}
      </div>

      <div className="grid gap-1.5">
        <Label htmlFor="settings-confirm-password">
          {t('settings.confirmPasswordLabel')}
        </Label>
        <Input
          id="settings-confirm-password"
          type="password"
          autoComplete="new-password"
          aria-invalid={errors.confirmPassword ? true : undefined}
          aria-describedby={
            errors.confirmPassword ? 'settings-confirm-password-error' : undefined
          }
          className="h-11"
          {...form.register('confirmPassword')}
        />
        {errors.confirmPassword ? (
          <p id="settings-confirm-password-error" className="m-0 text-xs text-red-600">
            {message(errors.confirmPassword.message)}
          </p>
        ) : null}
      </div>

      <div className="md:col-span-3">
        <Button type="submit" disabled={isSubmitting}>
          <KeyRound className="h-4 w-4" aria-hidden="true" />
          {t('settings.changePasswordButton')}
        </Button>
      </div>
    </form>
  )
}
