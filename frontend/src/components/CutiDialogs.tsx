import { useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm, useWatch } from 'react-hook-form'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'
import { createCuti, updateCuti, approveCuti, rejectCuti } from '@/lib/cuti'
import {
  cutiCreateSchema,
  cutiUpdateSchema,
  cutiActionSchema,
  type Cuti,
} from '@/validations/schemas'
import type { z } from 'zod'

const JENIS_OPTIONS = [
  { value: 'TAHUNAN', labelKey: 'cuti.jenisTahunan' },
  { value: 'PANJANG', labelKey: 'cuti.jenisPanjang' },
  { value: 'BERSALIN', labelKey: 'cuti.jenisBersalin' },
  { value: 'SAKIT', labelKey: 'cuti.jenisSakit' },
  { value: 'SAKIT_BERKEPANJANGAN', labelKey: 'cuti.jenisSakitBerkepanjangan' },
  { value: 'IZIN_PENTING', labelKey: 'cuti.jenisIzinPenting' },
  { value: 'POLITIK', labelKey: 'cuti.jenisPolitik' },
]

const SUBJENIS_OPTIONS = [
  { value: 'MENIKAH', labelKey: 'cuti.subMenikah' },
  { value: 'MELAHIRKAN_ISTRI', labelKey: 'cuti.subMelahirkanIstri' },
  { value: 'MENIKAHKAN_ANAK', labelKey: 'cuti.subMenikahkanAnak' },
  { value: 'KELUARGA_INTI', labelKey: 'cuti.subKeluargaInti' },
  { value: 'KELUARGA_LAIN', labelKey: 'cuti.subKeluargaLain' },
]

export function CreateCutiDialog({
  open,
  onClose,
  onSuccess,
}: {
  open: boolean
  onClose: () => void
  onSuccess: (cuti: Cuti) => void
}) {
  const t = useMessages()
  const form = useForm<z.infer<typeof cutiCreateSchema>>({
    resolver: zodResolver(cutiCreateSchema),
    defaultValues: {
      userId: '',
      jenis: 'TAHUNAN',
      tanggalMulai: '',
      tanggalSelesai: '',
      alasan: '',
    },
  })
  const jenis = useWatch({ control: form.control, name: 'jenis' })

  function handleSubmit(values: z.infer<typeof cutiCreateSchema>) {
    createCuti(values)
      .then(onSuccess)
      .catch(() => toast.error(t('cuti.createError')))
  }

  const isIzinPenting = jenis === 'IZIN_PENTING'

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>{t('cuti.createTitle')}</DialogTitle>
          <DialogDescription>{t('cuti.createDescription')}</DialogDescription>
        </DialogHeader>
        <form
          onSubmit={form.handleSubmit(handleSubmit)}
          className="grid gap-3"
          noValidate
        >
          <div>
            <Label htmlFor="create-cuti-userId">{t('cuti.userIdLabel')}</Label>
            <Input
              id="create-cuti-userId"
              {...form.register('userId')}
              className="h-11"
              placeholder="Masukkan ID pegawai (UUID)"
            />
            {form.formState.errors.userId ? (
              <p className="text-sm text-destructive mt-1">
                {form.formState.errors.userId.message}
              </p>
            ) : null}
          </div>

          <div>
            <Label htmlFor="create-cuti-jenis">{t('cuti.jenisLabel')}</Label>
            <Select
              value={jenis}
              onValueChange={(value) =>
                form.setValue('jenis', value as z.infer<typeof cutiCreateSchema>['jenis'])
              }
            >
              <SelectTrigger id="create-cuti-jenis" className="h-11">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {JENIS_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {t(option.labelKey as unknown as MessageKey)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label htmlFor="create-cuti-tanggalMulai">
              {t('cuti.tanggalMulaiLabel')}
            </Label>
            <Input
              id="create-cuti-tanggalMulai"
              type="date"
              {...form.register('tanggalMulai')}
              className="h-11"
            />
            {form.formState.errors.tanggalMulai ? (
              <p className="text-sm text-destructive mt-1">
                {form.formState.errors.tanggalMulai.message}
              </p>
            ) : null}
          </div>

          <div>
            <Label htmlFor="create-cuti-tanggalSelesai">
              {t('cuti.tanggalSelesaiLabel')}
            </Label>
            <Input
              id="create-cuti-tanggalSelesai"
              type="date"
              {...form.register('tanggalSelesai')}
              className="h-11"
            />
            {form.formState.errors.tanggalSelesai ? (
              <p className="text-sm text-destructive mt-1">
                {form.formState.errors.tanggalSelesai.message}
              </p>
            ) : null}
          </div>

          <div>
            <Label htmlFor="create-cuti-alasan">{t('cuti.alasanLabel')}</Label>
            <Input
              id="create-cuti-alasan"
              {...form.register('alasan')}
              className="h-11"
            />
            {form.formState.errors.alasan ? (
              <p className="text-sm text-destructive mt-1">
                {form.formState.errors.alasan.message}
              </p>
            ) : null}
          </div>

          {isIzinPenting && (
            <div>
              <Label htmlFor="create-cuti-subJenis">{t('cuti.subJenisLabel')}</Label>
              <Select {...form.register('subJenis')}>
                <SelectTrigger id="create-cuti-subJenis" className="h-11">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SUBJENIS_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {t(option.labelKey as unknown as MessageKey)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button type="submit">{t('common.save')}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function EditCutiDialog({
  cuti,
  onClose,
  onSuccess,
}: {
  cuti: Cuti
  onClose: () => void
  onSuccess: (cuti: Cuti) => void
}) {
  const t = useMessages()
  const form = useForm<z.infer<typeof cutiUpdateSchema>>({
    resolver: zodResolver(cutiUpdateSchema),
    defaultValues: {
      jenis: cuti.jenis,
      tanggalMulai: cuti.tanggalMulai,
      tanggalSelesai: cuti.tanggalSelesai,
      alamatCuti: cuti.alamatCuti ?? '',
      alasan: cuti.alasan ?? '',
    },
  })
  const jenis = useWatch({ control: form.control, name: 'jenis' })

  function handleSubmit(values: z.infer<typeof cutiUpdateSchema>) {
    const patch: Partial<z.infer<typeof cutiUpdateSchema>> = {}
    if (values.jenis && values.jenis !== cuti.jenis) patch.jenis = values.jenis
    if (values.tanggalMulai && values.tanggalMulai !== cuti.tanggalMulai)
      patch.tanggalMulai = values.tanggalMulai
    if (values.tanggalSelesai && values.tanggalSelesai !== cuti.tanggalSelesai)
      patch.tanggalSelesai = values.tanggalSelesai
    if (values.alamatCuti !== undefined && values.alamatCuti !== cuti.alamatCuti)
      patch.alamatCuti = values.alamatCuti
    if (values.alasan && values.alasan !== cuti.alasan) patch.alasan = values.alasan
    if (Object.keys(patch).length === 0) {
      onClose()
      return
    }
    updateCuti(cuti.id, patch)
      .then(onSuccess)
      .catch(() => toast.error(t('cuti.updateError')))
  }

  const isIzinPenting = jenis === 'IZIN_PENTING'

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>{t('cuti.editTitle')}</DialogTitle>
          <DialogDescription>{t('cuti.editDescription')}</DialogDescription>
        </DialogHeader>
        <form
          onSubmit={form.handleSubmit(handleSubmit)}
          className="grid gap-3"
          noValidate
        >
          <div>
            <Label htmlFor="edit-cuti-jenis">{t('cuti.jenisLabel')}</Label>
            <Select
              value={jenis}
              onValueChange={(value) =>
                form.setValue('jenis', value as z.infer<typeof cutiUpdateSchema>['jenis'])
              }
            >
              <SelectTrigger id="edit-cuti-jenis" className="h-11">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {JENIS_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {t(option.labelKey as unknown as MessageKey)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label htmlFor="edit-cuti-tanggalMulai">{t('cuti.tanggalMulaiLabel')}</Label>
            <Input
              id="edit-cuti-tanggalMulai"
              type="date"
              {...form.register('tanggalMulai')}
              className="h-11"
            />
          </div>

          <div>
            <Label htmlFor="edit-cuti-tanggalSelesai">
              {t('cuti.tanggalSelesaiLabel')}
            </Label>
            <Input
              id="edit-cuti-tanggalSelesai"
              type="date"
              {...form.register('tanggalSelesai')}
              className="h-11"
            />
          </div>

          <div>
            <Label htmlFor="edit-cuti-alamatCuti">{t('cuti.alamatCutiLabel')}</Label>
            <Input
              id="edit-cuti-alamatCuti"
              {...form.register('alamatCuti')}
              className="h-11"
            />
          </div>

          <div>
            <Label htmlFor="edit-cuti-alasan">{t('cuti.alasanLabel')}</Label>
            <Input id="edit-cuti-alasan" {...form.register('alasan')} className="h-11" />
          </div>

          {isIzinPenting && (
            <div>
              <Label htmlFor="edit-cuti-subJenis">{t('cuti.subJenisLabel')}</Label>
              <Select {...form.register('subJenis')}>
                <SelectTrigger id="edit-cuti-subJenis" className="h-11">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SUBJENIS_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {t(option.labelKey as unknown as MessageKey)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button type="submit">{t('common.save')}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function ApproveCutiDialog({
  cuti,
  onClose,
  onSuccess,
}: {
  cuti: Cuti
  onClose: () => void
  onSuccess: (cuti: Cuti) => void
}) {
  const t = useMessages()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const form = useForm<z.infer<typeof cutiActionSchema>>({
    resolver: zodResolver(cutiActionSchema),
    defaultValues: { catatan: '' },
  })

  function handleSubmit(values: z.infer<typeof cutiActionSchema>) {
    setIsSubmitting(true)
    approveCuti(cuti.id, { catatan: values.catatan ?? undefined })
      .then(onSuccess)
      .catch(() => {
        setIsSubmitting(false)
        toast.error(t('cuti.approveError'))
      })
  }

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>{t('cuti.approveTitle')}</DialogTitle>
          <DialogDescription>
            {t('cuti.approveDescription', { user: cuti.userId })}
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={form.handleSubmit(handleSubmit)}
          className="grid gap-3"
          noValidate
        >
          <div>
            <Label htmlFor="approve-cuti-catatan">Catatan (opsional)</Label>
            <Input
              id="approve-cuti-catatan"
              {...form.register('catatan')}
              className="h-11"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? t('common.saving') : t('cuti.approveButton')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function RejectCutiDialog({
  cuti,
  onClose,
  onSuccess,
}: {
  cuti: Cuti
  onClose: () => void
  onSuccess: (cuti: Cuti) => void
}) {
  const t = useMessages()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const form = useForm<z.infer<typeof cutiActionSchema>>({
    resolver: zodResolver(cutiActionSchema),
    defaultValues: { catatan: '' },
  })

  function handleSubmit(values: z.infer<typeof cutiActionSchema>) {
    setIsSubmitting(true)
    rejectCuti(cuti.id, { catatan: values.catatan ?? undefined })
      .then(onSuccess)
      .catch(() => {
        setIsSubmitting(false)
        toast.error(t('cuti.rejectError'))
      })
  }

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>{t('cuti.rejectTitle')}</DialogTitle>
          <DialogDescription>
            {t('cuti.rejectDescription', { user: cuti.userId })}
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={form.handleSubmit(handleSubmit)}
          className="grid gap-3"
          noValidate
        >
          <div>
            <Label htmlFor="reject-cuti-catatan">Alasan penolakan (opsional)</Label>
            <Input
              id="reject-cuti-catatan"
              {...form.register('catatan')}
              className="h-11"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" variant="destructive" disabled={isSubmitting}>
              {isSubmitting ? t('common.saving') : t('cuti.rejectButton')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
