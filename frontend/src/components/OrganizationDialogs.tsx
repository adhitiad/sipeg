import { useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
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
import { Textarea } from '@/components/ui/textarea'
import { useMessages } from '@/i18n/locale-context'
import type { MessageKey } from '@/i18n/dictionary'
import {
  archiveOrganization,
  createOrganization,
  ORGANIZATION_KINDS,
  updateOrganization,
} from '@/lib/organizations'
import {
  organizationCreateSchema,
  organizationUpdateSchema,
  type Organization,
  type OrganizationCreate,
  type OrganizationUpdate,
} from '@/validations/schemas'

const createSchema = organizationCreateSchema.extend({
  name: z.string().min(1, 'Nama unit wajib diisi').max(200),
})

const updateSchema = organizationUpdateSchema.extend({
  name: z.string().min(1, 'Nama unit wajib diisi').max(200).optional(),
})

type CreateFormValues = z.infer<typeof createSchema>
type UpdateFormValues = z.infer<typeof updateSchema>

function kindMessageKey(kind: string): string {
  return `organizations.type${kind.charAt(0)}${kind.slice(1).toLowerCase()}`
}

function kindOptions(t: (key: MessageKey) => string) {
  return ORGANIZATION_KINDS.map((kind: string) => (
    <SelectItem key={kind} value={kind}>
      {t(kindMessageKey(kind) as MessageKey)}
    </SelectItem>
  ))
}

export function CreateOrganizationDialog({
  open,
  onClose,
  onSuccess,
  parentUnits,
}: {
  open: boolean
  onClose: () => void
  onSuccess: (organization: Organization) => void
  parentUnits: Organization[]
}) {
  const t = useMessages()
  const form = useForm<CreateFormValues>({
    resolver: zodResolver(createSchema),
    defaultValues: {
      name: '',
      description: null,
      jenis: 'UNIT',
      parentId: null,
    },
  })
  const jenis = useWatch({ control: form.control, name: 'jenis' })
  const parentId = useWatch({ control: form.control, name: 'parentId' })

  function handleSubmit(values: CreateFormValues) {
    const payload: OrganizationCreate = {
      name: values.name,
      description: values.description,
      jenis: values.jenis ?? 'UNIT',
      parentId: values.parentId ?? null,
    }
    createOrganization(payload)
      .then(onSuccess)
      .catch(() => {
        toast.error(t('organizations.createError' as never))
      })
  }

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('organizations.createTitle' as never)}</DialogTitle>
          <DialogDescription>
            {t('organizations.createDescription' as never)}
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={form.handleSubmit(handleSubmit)}
          className="grid gap-3"
          noValidate
        >
          <div>
            <Label htmlFor="create-name">{t('organizations.nameLabel' as never)}</Label>
            <Input id="create-name" {...form.register('name')} className="h-11" />
            {form.formState.errors.name ? (
              <p className="text-sm text-destructive mt-1">
                {form.formState.errors.name.message}
              </p>
            ) : null}
          </div>
          <div>
            <Label htmlFor="create-jenis">{t('organizations.typeLabel' as never)}</Label>
            <Select
              value={jenis ?? 'UNIT'}
              onValueChange={(value) =>
                form.setValue('jenis', value as Organization['jenis'])
              }
            >
              <SelectTrigger id="create-jenis" className="h-11">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>{kindOptions(t)}</SelectContent>
            </Select>
          </div>
          <div>
            <Label htmlFor="create-parent">
              {t('organizations.parentLabel' as never)}
            </Label>
            <Select
              value={parentId ?? ''}
              onValueChange={(value) => form.setValue('parentId', value || null)}
            >
              <SelectTrigger id="create-parent" className="h-11">
                <SelectValue placeholder={t('organizations.noParent' as never)} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">{t('organizations.noParent' as never)}</SelectItem>
                {parentUnits.map((unit) => (
                  <SelectItem key={unit.id} value={unit.id}>
                    {unit.name ?? unit.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label htmlFor="create-description">
              {t('organizations.descriptionLabel' as never)}
            </Label>
            <Textarea
              id="create-description"
              {...form.register('description')}
              rows={3}
              placeholder={t('organizations.descriptionPlaceholder' as never)}
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('common.cancel' as never)}
            </Button>
            <Button type="submit">{t('common.save' as never)}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function EditOrganizationDialog({
  organization,
  onClose,
  onSuccess,
  parentUnits,
  parentNames,
}: {
  organization: Organization
  onClose: () => void
  onSuccess: (organization: Organization) => void
  parentUnits: Organization[]
  parentNames: Map<string, string>
}) {
  const t = useMessages()
  const form = useForm<UpdateFormValues>({
    resolver: zodResolver(updateSchema),
    defaultValues: {
      name: organization.name ?? '',
      description: null,
      jenis: organization.jenis,
      parentId: organization.parentId ?? null,
    },
  })
  const jenis = useWatch({ control: form.control, name: 'jenis' })
  const parentId = useWatch({ control: form.control, name: 'parentId' })

  function handleSubmit(values: UpdateFormValues) {
    const payload: OrganizationUpdate = {}
    if (values.name && values.name !== organization.name) payload.name = values.name
    if (values.jenis && values.jenis !== organization.jenis) payload.jenis = values.jenis
    if (values.parentId !== organization.parentId)
      payload.parentId = values.parentId ?? null

    updateOrganization(organization.id, payload)
      .then(onSuccess)
      .catch(() => {
        toast.error(t('organizations.updateError' as never))
      })
  }

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('organizations.editTitle' as never)}</DialogTitle>
          <DialogDescription>
            {t('organizations.editDescription' as never, {
              name: organization.name ?? '',
            })}
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={form.handleSubmit(handleSubmit)}
          className="grid gap-3"
          noValidate
        >
          <div>
            <Label htmlFor="edit-name">{t('organizations.nameLabel' as never)}</Label>
            <Input id="edit-name" {...form.register('name')} className="h-11" />
            {form.formState.errors.name ? (
              <p className="text-sm text-destructive mt-1">
                {form.formState.errors.name.message}
              </p>
            ) : null}
          </div>
          <div>
            <Label htmlFor="edit-jenis">{t('organizations.typeLabel' as never)}</Label>
            <Select
              value={jenis ?? 'UNIT'}
              onValueChange={(value) =>
                form.setValue('jenis', value as Organization['jenis'])
              }
            >
              <SelectTrigger id="edit-jenis" className="h-11">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>{kindOptions(t)}</SelectContent>
            </Select>
          </div>
          <div>
            <Label htmlFor="edit-parent">{t('organizations.parentLabel' as never)}</Label>
            <Select
              value={parentId ?? ''}
              onValueChange={(value) => form.setValue('parentId', value || null)}
            >
              <SelectTrigger id="edit-parent" className="h-11">
                <SelectValue placeholder={t('organizations.noParent' as never)} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">{t('organizations.noParent' as never)}</SelectItem>
                {parentUnits.map((unit) => (
                  <SelectItem key={unit.id} value={unit.id}>
                    {parentNames.get(unit.id) ?? unit.name ?? unit.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('common.cancel' as never)}
            </Button>
            <Button type="submit">{t('common.save' as never)}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function ArchiveOrganizationDialog({
  organization,
  onClose,
  onSuccess,
}: {
  organization: Organization
  onClose: () => void
  onSuccess: (name: string) => void
}) {
  const t = useMessages()
  const [isSubmitting, setIsSubmitting] = useState(false)

  function handleArchive() {
    setIsSubmitting(true)
    archiveOrganization(organization.id)
      .then(() => onSuccess(organization.name ?? organization.id))
      .catch(() => {
        setIsSubmitting(false)
        toast.error(t('organizations.archiveError' as never))
      })
  }

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('organizations.archiveTitle' as never)}</DialogTitle>
          <DialogDescription>
            {t('organizations.archiveDescription' as never, {
              name: organization.name ?? '',
            })}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t('common.cancel' as never)}
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleArchive}
            disabled={isSubmitting}
          >
            {t('organizations.archiveConfirm' as never)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
