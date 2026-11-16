import { useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { createFileRoute, Link } from '@tanstack/react-router'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { ChevronLeft, Edit2, Save, ShieldAlert, User } from 'lucide-react'
import RetryButton from '@/components/RetryButton'
import EmployeeRiwayatCards from '@/components/EmployeeRiwayatCards'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { useLocale, useMessages } from '@/i18n/locale-context'
import { type EmployeeUpdate } from '@/validations/schemas'
import { updateEmployee, useEmployee } from '@/lib/employees'
import { useOrganizations } from '@/lib/organizations'
import { useSession } from '@/lib/session'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import LoadingBars from '@/components/LoadingBars'

export const Route = createFileRoute('/_app/employees/$employeeId')({
  component: EmployeeProfilePage,
})

const profileSchema = z.object({
  nama: z.string().min(1, 'Nama wajib diisi').max(200),
  nip: z.string().max(50).nullable(),
  jabatanStruktural: z.string().max(200).nullable(),
  jabatanFungsional: z.string().max(200).nullable(),
})

export type ProfileFormValues = z.infer<typeof profileSchema>

function EmployeeProfilePage() {
  const t = useMessages()
  const { locale } = useLocale()
  const { employeeId } = Route.useParams()
  const queryClient = useQueryClient()
  const session = useSession()
  const units = useOrganizations()
  const [isEditing, setIsEditing] = useState(false)

  const permissions = session.data?.permissions ?? []
  const canUpdate =
    permissions.includes('employee:update:any') ||
    permissions.includes('employee:update:homebase')
  const canRead =
    permissions.includes('employee:read:any') ||
    permissions.includes('employee:read:homebase') ||
    permissions.includes('employee:read:self')

  const query = useEmployee(employeeId ?? '')

  const form = useForm<ProfileFormValues>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      nama: '',
      nip: null,
      jabatanStruktural: null,
      jabatanFungsional: null,
    },
  })

  const employee = query.data
  const unitNames = new Map<string, string>()
  for (const unit of units.data ?? []) {
    if (unit.name) {
      unitNames.set(unit.id, unit.name)
    }
  }

  if (session.data && !canRead) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <ShieldAlert
            className="mx-auto mb-3 h-8 w-8 text-[var(--sea-ink-soft)]"
            aria-hidden="true"
          />
          <h1 className="display-title mb-2 text-2xl font-bold text-[var(--sea-ink)]">
            {t('state.noAccess.title')}
          </h1>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('state.noAccess.description')}
          </p>
        </section>
      </div>
    )
  }

  if (query.isPending || !employee) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <div className="island-shell flex flex-col items-center gap-4 rounded-2xl px-6 py-12 text-center">
          <LoadingBars size="md" decorative />
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('employees.loadingTitle')}
          </p>
        </div>
      </div>
    )
  }

  if (query.isError || !employee) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('employees.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('employees.errorDescription')}
          </p>
          <RetryButton label={t('common.retry')} onClick={() => void query.refetch()} />
        </div>
      </div>
    )
  }

  if (!employee) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <h1 className="display-title mb-2 text-2xl font-bold text-[var(--sea-ink)]">
            {t('state.notFound.title')}
          </h1>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('state.notFound.description')}
          </p>
        </section>
      </div>
    )
  }

  const localeTag = locale === 'id' ? 'id-ID' : 'en-US'

  function handleSubmit(values: ProfileFormValues) {
    const payload: EmployeeUpdate = {}
    if (values.nama !== employee!.nama) payload.nama = values.nama
    if (values.nip !== employee!.nip) payload.nip = values.nip
    if (values.jabatanStruktural !== employee!.jabatanStruktural)
      payload.jabatanStruktural = values.jabatanStruktural
    if (values.jabatanFungsional !== employee!.jabatanFungsional)
      payload.jabatanFungsional = values.jabatanFungsional

    updateEmployee(employee!.id, payload).then((updated) => {
      form.reset({
        nama: updated.nama,
        nip: updated.nip,
        jabatanStruktural: updated.jabatanStruktural,
        jabatanFungsional: updated.jabatanFungsional,
      })
      setIsEditing(false)
      queryClient.invalidateQueries({ queryKey: ['employee', updated.id] })
      queryClient.invalidateQueries({ queryKey: ['employees'] })
      toast.success(t('login.success'))
    })
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
        <div>
          <p className="island-kicker mb-2">{t('employees.kicker')}</p>
          <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
            {t('employees.profileTitle')}
          </h1>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('employees.profileLead', { nama: employee.nama })}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Link to="/employees" className="nav-link">
            <Button variant="outline" size="sm" className="flex items-center gap-2">
              <ChevronLeft className="h-4 w-4" aria-hidden="true" />
              {t('employees.backToList')}
            </Button>
          </Link>
          {canUpdate && (
            <Button
              onClick={() => {
                form.reset({
                  nama: employee.nama,
                  nip: employee.nip,
                  jabatanStruktural: employee.jabatanStruktural,
                  jabatanFungsional: employee.jabatanFungsional,
                })
                setIsEditing(true)
              }}
              disabled={isEditing}
              className="flex items-center gap-2"
            >
              {isEditing ? (
                <Save className="h-4 w-4" aria-hidden="true" />
              ) : (
                <Edit2 className="h-4 w-4" aria-hidden="true" />
              )}
              <span>{isEditing ? t('common.saving') : t('common.edit')}</span>
            </Button>
          )}
        </div>
      </div>

      <Card className="island-shadow max-w-2xl">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <User className="h-5 w-5" aria-hidden="true" />
            {t('employees.profileDetails')}
          </CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <div className="col-span-full sm:col-span-2">
            <Separator />
          </div>
          <div className="lg:col-span-2">
            <Label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('employees.colName')}
            </Label>
            <p className="text-[var(--sea-ink)] font-medium">{employee.nama}</p>
          </div>
          <div className="lg:col-span-2">
            <Label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('employees.fieldEmail')}
            </Label>
            <p className="font-mono text-xs break-all text-[var(--sea-ink-soft)]">
              {employee.email}
            </p>
          </div>
          <div>
            <Label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('employees.fieldNip')}
            </Label>
            <p className="font-mono text-xs text-[var(--sea-ink)]">
              {employee.nip ?? t('employees.noValue')}
            </p>
          </div>
          <div>
            <Label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('employees.fieldUnit')}
            </Label>
            <p className="text-[var(--sea-ink)] text-sm">
              {employee.homebaseId
                ? (unitNames.get(employee.homebaseId) ?? employee.homebaseId)
                : t('employees.noUnit')}
            </p>
          </div>
          <div className="lg:col-span-2">
            <Label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('employees.fieldStructuralPosition')}
            </Label>
            <p className="text-[var(--sea-ink)]">
              {employee.jabatanStruktural ?? t('employees.noValue')}
            </p>
          </div>
          <div className="lg:col-span-2">
            <Label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('employees.fieldFunctionalPosition')}
            </Label>
            <p className="text-[var(--sea-ink)]">
              {employee.jabatanFungsional ?? t('employees.noValue')}
            </p>
          </div>
          <div>
            <Label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('employees.fieldStatus')}
            </Label>
            <p
              className={`font-medium text-sm ${employee.aktif ? 'text-[var(--sea-ink)]' : 'text-destructive'}`}
            >
              {employee.aktif
                ? t('employees.statusActive')
                : t('employees.statusInactive')}
            </p>
          </div>
          <div>
            <Label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('employees.fieldRole')}
            </Label>
            <p className="text-[var(--sea-ink-soft)] text-sm">
              {session.data?.roleName ?? t('role.missing')}
            </p>
          </div>
          <div className="col-span-full">
            <Separator />
          </div>
          <div className="lg:col-span-2">
            <Label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('employees.fieldCreatedAt')}
            </Label>
            <p className="font-mono text-xs text-[var(--sea-ink-soft)] break-all">
              {new Date(employee.createdAt).toLocaleString(localeTag)}
            </p>
          </div>
          <div className="lg:col-span-2">
            <Label className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('employees.fieldUpdatedAt')}
            </Label>
            <p className="font-mono text-xs text-[var(--sea-ink-soft)] break-all">
              {new Date(employee.updatedAt).toLocaleString(localeTag)}
            </p>
          </div>
        </CardContent>
      </Card>

      <EmployeeRiwayatCards userId={employeeId} />

      {canUpdate && isEditing ? (
        <Card className="mt-6 max-w-md">
          <CardHeader>
            <CardTitle>{t('employees.editProfile')}</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              onSubmit={form.handleSubmit(handleSubmit)}
              className="grid gap-3"
              noValidate
            >
              <div>
                <Label htmlFor="profile-nama">{t('employees.colName')}</Label>
                <Input
                  id="profile-nama"
                  {...form.register('nama')}
                  disabled={query.isPending}
                  className="h-11"
                />
                {form.formState.errors.nama ? (
                  <p className="text-sm text-destructive mt-1">
                    {form.formState.errors.nama.message}
                  </p>
                ) : null}
              </div>
              <div>
                <Label htmlFor="profile-nip">{t('employees.fieldNip')}</Label>
                <Input
                  id="profile-nip"
                  type="text"
                  {...form.register('nip')}
                  placeholder={t('employees.noValue')}
                  disabled={query.isPending}
                  className="h-11"
                />
              </div>
              <div>
                <Label htmlFor="profile-jabatanStruktural">
                  {t('employees.fieldStructuralPosition')}
                </Label>
                <Input
                  id="profile-jabatanStruktural"
                  {...form.register('jabatanStruktural')}
                  placeholder={t('employees.noValue')}
                  disabled={query.isPending}
                  className="h-11"
                />
              </div>
              <div>
                <Label htmlFor="profile-jabatanFungsional">
                  {t('employees.fieldFunctionalPosition')}
                </Label>
                <Input
                  id="profile-jabatanFungsional"
                  {...form.register('jabatanFungsional')}
                  placeholder={t('employees.noValue')}
                  disabled={query.isPending}
                  className="h-11"
                />
              </div>
              <div className="flex items-center gap-2 pt-2">
                <Button
                  type="submit"
                  disabled={query.isPending || !form.formState.isDirty}
                >
                  <Save className="h-4 w-4 mr-2" aria-hidden="true" />
                  {t('common.save')}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setIsEditing(false)}
                  disabled={query.isPending}
                >
                  {t('common.cancel')}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}
