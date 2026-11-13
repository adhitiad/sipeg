import { useRef, useState } from 'react'
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
import { archiveJabatan, createJabatan, updateJabatan } from '@/lib/jabatan'
import {
  jabatanCreateSchema,
  jabatanUpdateSchema,
  type Jabatan,
} from '@/validations/schemas'
import type { z } from 'zod'

const createSchema = jabatanCreateSchema
const updateSchema = jabatanUpdateSchema

type CreateFormValues = z.infer<typeof createSchema>
type UpdateFormValues = z.infer<typeof updateSchema>

export function CreateJabatanDialog({
  open,
  onClose,
  onSuccess,
}: {
  open: boolean
  onClose: () => void
  onSuccess: (jabatan: Jabatan) => void
}) {
  const t = useMessages()
  const form = useForm<CreateFormValues>({
    resolver: zodResolver(createSchema),
    defaultValues: { nama: '', jenis: 'STRUKTURAL' },
  })
  const jenis = useWatch({ control: form.control, name: 'jenis' })

  function handleSubmit(values: CreateFormValues) {
    createJabatan(values)
      .then(onSuccess)
      .catch(() => toast.error(t('jabatan.createError')))
  }

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('jabatan.createTitle')}</DialogTitle>
          <DialogDescription>{t('jabatan.createDescription')}</DialogDescription>
        </DialogHeader>
        <form
          onSubmit={form.handleSubmit(handleSubmit)}
          className="grid gap-3"
          noValidate
        >
          <div>
            <Label htmlFor="create-jabatan-nama">{t('jabatan.nameLabel')}</Label>
            <Input id="create-jabatan-nama" {...form.register('nama')} className="h-11" />
            {form.formState.errors.nama ? (
              <p className="text-sm text-destructive mt-1">
                {form.formState.errors.nama.message}
              </p>
            ) : null}
          </div>
          <div>
            <Label htmlFor="create-jabatan-jenis">{t('jabatan.typeLabel')}</Label>
            <Select
              value={jenis ?? 'STRUKTURAL'}
              onValueChange={(value) =>
                form.setValue('jenis', value as CreateFormValues['jenis'])
              }
            >
              <SelectTrigger id="create-jabatan-jenis" className="h-11">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="STRUKTURAL">{t('jabatan.typeStruktural')}</SelectItem>
                <SelectItem value="FUNGSIONAL">{t('jabatan.typeFungsional')}</SelectItem>
              </SelectContent>
            </Select>
          </div>
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

export function EditJabatanDialog({
  jabatan,
  onClose,
  onSuccess,
}: {
  jabatan: Jabatan
  onClose: () => void
  onSuccess: (jabatan: Jabatan) => void
}) {
  const t = useMessages()
  const form = useForm<UpdateFormValues>({
    resolver: zodResolver(updateSchema),
    defaultValues: { nama: jabatan.nama, jenis: jabatan.jenis },
  })
  const jenis = useWatch({ control: form.control, name: 'jenis' })

  function handleSubmit(values: UpdateFormValues) {
    const patch: { nama?: string; jenis?: 'STRUKTURAL' | 'FUNGSIONAL' } = {}
    if (values.nama && values.nama !== jabatan.nama) {
      patch.nama = values.nama
    }
    if (values.jenis && values.jenis !== jabatan.jenis) {
      patch.jenis = values.jenis
    }
    if (Object.keys(patch).length === 0) {
      onClose()
      return
    }
    updateJabatan(jabatan.id, patch)
      .then(onSuccess)
      .catch(() => toast.error(t('jabatan.updateError')))
  }

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('jabatan.editTitle')}</DialogTitle>
          <DialogDescription>
            {t('jabatan.editDescription', { name: jabatan.nama })}
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={form.handleSubmit(handleSubmit)}
          className="grid gap-3"
          noValidate
        >
          <div>
            <Label htmlFor="edit-jabatan-nama">{t('jabatan.nameLabel')}</Label>
            <Input id="edit-jabatan-nama" {...form.register('nama')} className="h-11" />
            {form.formState.errors.nama ? (
              <p className="text-sm text-destructive mt-1">
                {form.formState.errors.nama.message}
              </p>
            ) : null}
          </div>
          <div>
            <Label htmlFor="edit-jabatan-jenis">{t('jabatan.typeLabel')}</Label>
            <Select
              value={jenis ?? 'STRUKTURAL'}
              onValueChange={(value) =>
                form.setValue('jenis', value as UpdateFormValues['jenis'])
              }
            >
              <SelectTrigger id="edit-jabatan-jenis" className="h-11">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="STRUKTURAL">{t('jabatan.typeStruktural')}</SelectItem>
                <SelectItem value="FUNGSIONAL">{t('jabatan.typeFungsional')}</SelectItem>
              </SelectContent>
            </Select>
          </div>
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

export function ArchiveJabatanDialog({
  jabatan,
  onClose,
  onSuccess,
}: {
  jabatan: Jabatan
  onClose: () => void
  onSuccess: (name: string) => void
}) {
  const t = useMessages()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const cancelRef = useRef<HTMLButtonElement>(null)

  function handleArchive() {
    setIsSubmitting(true)
    archiveJabatan(jabatan.id)
      .then(() => onSuccess(jabatan.nama))
      .catch(() => {
        setIsSubmitting(false)
        toast.error(t('jabatan.archiveError'))
      })
  }

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('jabatan.archiveTitle')}</DialogTitle>
          <DialogDescription>
            {t('jabatan.archiveDescription', { name: jabatan.nama })}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button ref={cancelRef} type="button" variant="outline" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleArchive}
            disabled={isSubmitting}
          >
            {t('jabatan.archiveConfirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
