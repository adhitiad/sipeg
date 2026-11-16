import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil } from 'lucide-react'
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
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import { useMessages } from '@/i18n/locale-context'
import {
  createHonorRate,
  honorSksQueryKey,
  updateHonorRate,
  useHonorRates,
} from '@/lib/honor-sks'
import { honorSksRateCreateSchema, type HonorSksRate } from '@/validations/schemas'

type HonorRateSectionProps = {
  canManage: boolean
}

export default function HonorRateSection({ canManage }: HonorRateSectionProps) {
  const t = useMessages()
  const queryClient = useQueryClient()
  const rates = useHonorRates()
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [editing, setEditing] = useState<HonorSksRate | null>(null)
  const [formError, setFormError] = useState<string | null>(null)

  const saveMutation = useMutation({
    mutationFn: (values: { berlakuMulai: string; perSks: number }) => {
      const parsed = honorSksRateCreateSchema.safeParse(values)
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0]?.message ?? t('honorSks.rateSaveError'))
      }
      if (editing) {
        return updateHonorRate(editing.id, parsed.data)
      }
      return createHonorRate(parsed.data)
    },
    onSuccess: () => {
      toast.success(t('honorSks.rateSaved'))
      setIsFormOpen(false)
      setEditing(null)
      setFormError(null)
      void queryClient.invalidateQueries({ queryKey: honorSksQueryKey })
    },
    onError: (error: Error) => {
      setFormError(error.message)
    },
  })

  const toggleMutation = useMutation({
    mutationFn: (row: HonorSksRate) => updateHonorRate(row.id, { aktif: !row.aktif }),
    onSuccess: () => {
      toast.success(t('honorSks.rateSaved'))
      void queryClient.invalidateQueries({ queryKey: honorSksQueryKey })
    },
    onError: () => {
      toast.error(t('honorSks.rateSaveError'))
    },
  })

  if (rates.isPending) {
    return (
      <div className="py-8 text-center">
        <LoadingBars size="sm" label={t('honorSks.ratesTitle')} />
      </div>
    )
  }

  if (rates.isError || !rates.data) {
    return (
      <div className="py-8 text-center">
        <p className="mb-3 text-sm text-[var(--sea-ink-soft)]">
          {t('honorSks.rateError')}
        </p>
        <RetryButton label={t('common.retry')} onClick={() => rates.refetch()} />
      </div>
    )
  }

  const items = rates.data

  return (
    <div className="py-2">
      <div className="mb-3 flex items-center justify-between gap-3">
        <div>
          <h2 className="m-0 text-lg font-semibold text-[var(--sea-ink)]">
            {t('honorSks.ratesTitle')}
          </h2>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('honorSks.ratesLead')}
          </p>
        </div>
        {canManage ? (
          <Button
            type="button"
            onClick={() => {
              setEditing(null)
              setFormError(null)
              setIsFormOpen(true)
            }}
          >
            {t('honorSks.addRate')}
          </Button>
        ) : null}
      </div>

      {items.length === 0 ? (
        <p className="m-0 py-6 text-center text-sm text-[var(--sea-ink-soft)]">
          {t('honorSks.rateEmpty')}
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-[var(--line)]">
          <table className="min-w-full border-collapse text-sm">
            <caption className="sr-only">{t('honorSks.ratesTitle')}</caption>
            <thead>
              <tr className="text-left text-xs uppercase tracking-[0.16em] text-[var(--kicker)]">
                <th scope="col" className="px-3 py-2">
                  {t('honorSks.rateColBerlaku')}
                </th>
                <th scope="col" className="px-3 py-2 text-right">
                  {t('honorSks.rateColPerSks')}
                </th>
                <th scope="col" className="px-3 py-2">
                  {t('honorSks.rateColAktif')}
                </th>
                {canManage ? (
                  <th scope="col" className="px-3 py-2 text-right">
                    <span className="sr-only">Aksi</span>
                  </th>
                ) : null}
              </tr>
            </thead>
            <tbody>
              {items.map((row) => (
                <tr key={row.id} className="border-t border-[var(--line)] align-middle">
                  <td className="px-3 py-2 font-medium">
                    {new Date(row.berlakuMulai).toLocaleDateString('id-ID', {
                      day: '2-digit',
                      month: 'short',
                      year: 'numeric',
                    })}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    Rp {row.perSks.toLocaleString('id-ID')}
                  </td>
                  <td className="px-3 py-2">
                    <span
                      className={
                        row.aktif
                          ? 'rounded-full bg-[var(--lagoon)] px-2 py-0.5 text-xs font-bold text-white'
                          : 'rounded-full bg-[var(--sand)] px-2 py-0.5 text-xs font-semibold text-[var(--sea-ink-soft)]'
                      }
                    >
                      {row.aktif ? 'Ya' : 'Tidak'}
                    </span>
                  </td>
                  {canManage ? (
                    <td className="px-3 py-2 text-right">
                      <div className="flex justify-end gap-1.5">
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          aria-label={t('honorSks.rateEditTitle')}
                          onClick={() => {
                            setEditing(row)
                            setFormError(null)
                            setIsFormOpen(true)
                          }}
                        >
                          <Pencil className="h-4 w-4" aria-hidden="true" />
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="h-8 px-2 text-xs"
                          onClick={() => toggleMutation.mutate(row)}
                        >
                          {row.aktif ? t('common.deactivate') : t('common.activate')}
                        </Button>
                      </div>
                    </td>
                  ) : null}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Dialog
        open={isFormOpen}
        onOpenChange={(open) => {
          setIsFormOpen(open)
          if (!open) {
            setEditing(null)
            setFormError(null)
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editing ? t('honorSks.rateEditTitle') : t('honorSks.rateDialogTitle')}
            </DialogTitle>
            <DialogDescription>{t('honorSks.ratesLead')}</DialogDescription>
          </DialogHeader>
          <form
            className="grid gap-4"
            noValidate
            onSubmit={(event) => {
              event.preventDefault()
              const data = Object.fromEntries(
                new FormData(event.currentTarget).entries(),
              ) as Record<string, string>
              saveMutation.mutate({
                berlakuMulai: data.berlakuMulai?.trim() ?? '',
                perSks: Number(data.perSks),
              })
            }}
          >
            <div className="grid gap-1.5">
              <Label htmlFor="berlakuMulai">{t('honorSks.rateColBerlaku')}</Label>
              <Input
                id="berlakuMulai"
                name="berlakuMulai"
                type="date"
                required
                defaultValue={editing?.berlakuMulai?.slice(0, 10) ?? ''}
                className="h-11"
              />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="perSks">{t('honorSks.rateColPerSks')}</Label>
              <Input
                id="perSks"
                name="perSks"
                type="number"
                min={0}
                step="1"
                required
                defaultValue={editing?.perSks ?? ''}
                className="h-11"
              />
            </div>
            {formError ? (
              <p role="alert" className="m-0 text-sm text-destructive">
                {formError}
              </p>
            ) : null}
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsFormOpen(false)}
              >
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={saveMutation.isPending}>
                {saveMutation.isPending
                  ? t('payrollScale.saving')
                  : t('payrollScale.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
