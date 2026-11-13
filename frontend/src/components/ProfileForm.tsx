import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { User } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'
import { useUpdateProfile } from '@/lib/session'
import { updateProfileSchema } from '@/validations/schemas'

export default function ProfileForm() {
  const t = useMessages()
  const mutation = useUpdateProfile()
  const form = useForm({
    resolver: zodResolver(updateProfileSchema),
    defaultValues: { nama: '', noHp: '' },
  })
  const { errors } = form.formState
  const message = (key: string | undefined) => (key ? t(key as MessageKey) : null)

  async function onSubmit(values: { nama: string; noHp: string }) {
    try {
      await mutation.mutateAsync(values)
      toast.success(t('settings.profileSaved'))
    } catch {
      toast.error(t('settings.profileError'))
    }
  }

  return (
    <form className="grid gap-4" noValidate onSubmit={form.handleSubmit(onSubmit)}>
      <div className="grid gap-1.5 flex items-center gap-3">
        <User className="h-5 w-5 text-[var(--kicker)]" aria-hidden="true" />
        <div className="grid gap-1.5">
          <Label htmlFor="settings-profile-nama">{t('settings.profileNameLabel')}</Label>
          <Input id="settings-profile-nama" className="h-11" {...form.register('nama')} />
          {errors.nama ? (
            <p className="m-0 text-xs text-red-600">{message(errors.nama.message)}</p>
          ) : null}
        </div>
      </div>

      <div className="grid gap-1.5 flex items-center gap-3 md:flex-row md:gap-4">
        <div className="hidden md:flex items-center">
          <User className="h-5 w-5 text-[var(--kicker)]" aria-hidden="true" />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="settings-profile-noHp">{t('settings.noHpLabel')}</Label>
          <Input
            id="settings-profile-noHp"
            type="tel"
            inputMode="tel"
            className="h-11"
            {...form.register('noHp')}
          />
          <p className="m-0 text-xs text-[var(--kicker-soft)]">
            {t('settings.noHpHint')}
          </p>
          {errors.noHp ? (
            <p className="m-0 text-xs text-red-600">{message(errors.noHp.message)}</p>
          ) : null}
        </div>
      </div>

      <div className="md:flex md:justify-end">
        <Button type="submit" disabled={mutation.isPending}>
          {mutation.isPending ? t('common.saving') : t('common.save')}
        </Button>
      </div>
    </form>
  )
}
