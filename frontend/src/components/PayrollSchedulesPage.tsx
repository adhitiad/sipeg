import { useMemo, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { CalendarClock } from 'lucide-react'
import { toast } from 'sonner'
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
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
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useMessages } from '@/i18n/locale-context'
import {
  payrollSchedulesQueryKey,
  upsertPayrollSchedule,
  usePayrollSchedules,
} from '@/lib/payroll'
import { useOrganizations } from '@/lib/organizations'
import { useSession } from '@/lib/session'
import type { PayrollSchedule } from '@/validations/schemas'

export default function PayrollSchedulesPage() {
  const t = useMessages()
  const session = useSession()
  const queryClient = useQueryClient()
  const permissions = session.data?.permissions ?? []
  const canRead = permissions.some((permission) =>
    permission.startsWith('payroll:schedule'),
  )
  const canReadUnits =
    permissions.includes('organization:read') ||
    permissions.includes('organization:read:homebase')

  const schedules = usePayrollSchedules()
  const units = useOrganizations(canRead && canReadUnits)
  const [editing, setEditing] = useState<PayrollSchedule | null>(null)
  const [day, setDay] = useState('1')

  const unitNames = useMemo(() => {
    const map = new Map<string, string>()
    for (const unit of units.data ?? []) {
      if (unit.name) map.set(unit.id, unit.name)
    }
    return map
  }, [units.data])

  const mutation = useMutation({
    mutationFn: (payload: { unitId: string; tanggalGajian: number }) =>
      upsertPayrollSchedule(payload.unitId, payload.tanggalGajian),
    onSuccess: () => {
      toast.success(t('payroll.scheduleSuccess'))
      setEditing(null)
      void queryClient.invalidateQueries({ queryKey: payrollSchedulesQueryKey })
    },
    onError: () => {
      toast.error(t('payroll.scheduleError'))
    },
  })

  if (session.data && !canRead) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <CalendarClock
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

  if (schedules.isPending) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <LoadingBars label={t('payroll.schedulesLoadingTitle')} />
          <h1 className="display-title mt-3 text-2xl font-bold text-[var(--sea-ink)]">
            {t('payroll.schedulesLoadingTitle')}
          </h1>
        </section>
      </div>
    )
  }

  if (schedules.isError) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <h1 className="display-title mb-2 text-2xl font-bold text-[var(--sea-ink)]">
            {t('payroll.schedulesErrorTitle')}
          </h1>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('payroll.schedulesErrorDescription')}
          </p>
          <div className="mt-4 flex justify-center">
            <RetryButton label={t('common.retry')} onClick={() => schedules.refetch()} />
          </div>
        </section>
      </div>
    )
  }

  const items = schedules.data ?? []

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <section className="island-shell rise-in rounded-2xl p-6 sm:p-8">
        <p className="island-kicker mb-2">{t('payroll.kicker')}</p>
        <h1 className="display-title mb-2 text-3xl font-bold text-[var(--sea-ink)] sm:text-4xl">
          {t('payroll.schedulesTitle')}
        </h1>
        <p className="m-0 mb-6 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)]">
          {t('payroll.schedulesLead')}
        </p>

        {items.length === 0 ? (
          <p className="m-0 py-8 text-center text-sm text-[var(--sea-ink-soft)]">
            {t('payroll.schedulesEmpty')}
          </p>
        ) : (
          <Table className="min-w-[40rem]">
            <TableCaption className="sr-only">
              {t('payroll.scheduleCaption')}
            </TableCaption>
            <TableHeader>
              <TableRow className="border-[var(--line)] hover:bg-transparent">
                <TableHead scope="col" className="px-4 py-3">
                  {t('payroll.colUnit')}
                </TableHead>
                <TableHead scope="col" className="px-4 py-3">
                  {t('payroll.colPayDay')}
                </TableHead>
                <TableHead scope="col" className="px-4 py-3">
                  {t('payroll.colUpdatedAt')}
                </TableHead>
                <TableHead scope="col" className="px-4 py-3 text-right">
                  <span className="sr-only">Aksi</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((schedule) => {
                const unitName =
                  schedule.unitNama || unitNames.get(schedule.unitId) || schedule.unitId
                return (
                  <TableRow
                    key={schedule.unitId}
                    className="border-[var(--line)] last:border-0"
                  >
                    <TableCell className="px-4 py-3 font-medium text-[var(--sea-ink)]">
                      {unitName}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-[var(--sea-ink)]">
                      {schedule.tanggalGajian}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-[var(--sea-ink-soft)] text-sm">
                      {new Date(schedule.updatedAt).toLocaleString()}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          setEditing(schedule)
                          setDay(String(schedule.tanggalGajian))
                        }}
                      >
                        {t('payroll.editSchedule')}
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        )}
      </section>

      <Dialog open={!!editing} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('payroll.scheduleDialogTitle')}</DialogTitle>
            <DialogDescription>
              {t('payroll.scheduleDialogDescription', {
                unit:
                  editing?.unitNama ||
                  (editing ? unitNames.get(editing.unitId) || editing.unitId : ''),
              })}
            </DialogDescription>
          </DialogHeader>
          <form
            className="grid gap-3"
            onSubmit={(event) => {
              event.preventDefault()
              if (!editing) return
              const value = Number(day)
              mutation.mutate({ unitId: editing.unitId, tanggalGajian: value })
            }}
          >
            <label className="grid gap-1">
              <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
                {t('payroll.payDayLabel')}
              </span>
              <Input
                type="number"
                min={1}
                max={28}
                value={day}
                onChange={(event) => setDay(event.target.value)}
                required
                className="h-11"
              />
            </label>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditing(null)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {t('payroll.scheduleSave')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
