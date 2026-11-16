import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Trash2 } from 'lucide-react'
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
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import { useMessages } from '@/i18n/locale-context'
import {
  createPayrollScaleDetail,
  deletePayrollScaleDetail,
  updatePayrollScaleDetail,
  useKategoriStatus,
  usePayrollScaleDetails,
} from '@/lib/payroll-scale'
import {
  payrollScaleDetailCreateSchema,
  payrollScaleDetailUpdateSchema,
  type PayrollScale,
  type PayrollScaleDetail,
  type PayrollScaleDetailCreate,
} from '@/validations/schemas'

type PayrollScaleDetailDialogProps = {
  scale: PayrollScale | null
  open: boolean
  onOpenChange: (open: boolean) => void
  canManage: boolean
}

type FormState = {
  kategoriStatusId: string
  tingkat: string
  nominal: string
}

const EMPTY_FORM: FormState = { kategoriStatusId: '', tingkat: '0', nominal: '' }

export default function PayrollScaleDetailDialog({
  scale,
  open,
  onOpenChange,
  canManage,
}: PayrollScaleDetailDialogProps) {
  const t = useMessages()
  const queryClient = useQueryClient()
  const details = usePayrollScaleDetails(scale?.id ?? null)
  const kategori = useKategoriStatus()
  const [formError, setFormError] = useState<string | null>(null)
  const [editing, setEditing] = useState<PayrollScaleDetail | null>(null)
  const [form, setForm] = useState<FormState>(EMPTY_FORM)

  const handleOpenChange = (next: boolean) => {
    if (!next) {
      setEditing(null)
      setForm(EMPTY_FORM)
      setFormError(null)
    }
    onOpenChange(next)
  }

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['payroll-scale'] })

  const saveMutation = useMutation({
    mutationFn: async (values: FormState): Promise<unknown> => {
      const tingkat = Number(values.tingkat)
      const nominal = Number(values.nominal)
      if (editing) {
        const parsed = payrollScaleDetailUpdateSchema.safeParse({
          kategoriStatusId: values.kategoriStatusId || undefined,
          tingkat: Number.isFinite(tingkat) ? tingkat : undefined,
          nominal: Number.isFinite(nominal) ? nominal : undefined,
        })
        if (!parsed.success) {
          throw new Error(
            parsed.error.issues[0]?.message ?? t('payrollScale.detailError'),
          )
        }
        return updatePayrollScaleDetail(editing.id, parsed.data)
      }
      const parsed = payrollScaleDetailCreateSchema.safeParse({
        kategoriStatusId: values.kategoriStatusId,
        tingkat: Number.isFinite(tingkat) ? tingkat : NaN,
        nominal: Number.isFinite(nominal) ? nominal : NaN,
      })
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0]?.message ?? t('payrollScale.detailError'))
      }
      const payload: PayrollScaleDetailCreate = parsed.data
      return createPayrollScaleDetail(scale!.id, payload)
    },
    onSuccess: () => {
      toast.success(t('payrollScale.detailSaved'))
      setForm(EMPTY_FORM)
      setEditing(null)
      setFormError(null)
      void invalidate()
    },
    onError: (error: Error) => {
      setFormError(error.message)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deletePayrollScaleDetail(id),
    onSuccess: () => {
      toast.success(t('payrollScale.detailDeleted'))
      void invalidate()
    },
    onError: () => {
      toast.error(t('payrollScale.detailError'))
    },
  })

  const rows = details.data?.details ?? []
  const rupiah = new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  })

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t('payrollScale.detailTitle')}</DialogTitle>
          <DialogDescription>{t('payrollScale.detailLead')}</DialogDescription>
        </DialogHeader>

        {details.isPending ? (
          <div className="py-6 text-center">
            <LoadingBars size="sm" label={t('payrollScale.loadingTitle')} />
          </div>
        ) : details.isError ? (
          <div className="py-6 text-center">
            <RetryButton
              label={t('common.retry')}
              onClick={() => void details.refetch()}
            />
          </div>
        ) : (
          <>
            <div className="max-h-72 overflow-auto rounded-xl border border-[var(--line)]">
              <table className="min-w-full border-collapse text-sm">
                <caption className="sr-only">{t('payrollScale.detailTitle')}</caption>
                <thead className="sticky top-0 bg-[var(--chip-bg)]">
                  <tr className="text-left text-xs uppercase tracking-[0.16em] text-[var(--kicker)]">
                    <th scope="col" className="px-3 py-2">
                      {t('payrollScale.detailKategori')}
                    </th>
                    <th scope="col" className="px-3 py-2">
                      {t('payrollScale.detailTingkat')}
                    </th>
                    <th scope="col" className="px-3 py-2 text-right">
                      {t('payrollScale.detailNominal')}
                    </th>
                    {canManage ? (
                      <th scope="col" className="px-3 py-2 text-right">
                        <span className="sr-only">Aksi</span>
                      </th>
                    ) : null}
                  </tr>
                </thead>
                <tbody>
                  {rows.length === 0 ? (
                    <tr>
                      <td
                        colSpan={canManage ? 4 : 3}
                        className="px-3 py-6 text-center text-sm text-[var(--sea-ink-soft)]"
                      >
                        {t('payrollScale.detailEmpty')}
                      </td>
                    </tr>
                  ) : (
                    rows.map((row) => (
                      <tr
                        key={row.id}
                        className="border-t border-[var(--line)] align-middle text-[var(--sea-ink)]"
                      >
                        <td className="px-3 py-2">
                          <span className="block font-medium">{row.kategoriNama}</span>
                          <span className="block text-xs text-[var(--sea-ink-soft)]">
                            {row.kategoriKode}
                          </span>
                        </td>
                        <td className="px-3 py-2 tabular-nums">{row.tingkat}</td>
                        <td className="px-3 py-2 text-right tabular-nums">
                          {rupiah.format(row.nominal)}
                        </td>
                        {canManage ? (
                          <td className="px-3 py-2 text-right">
                            <div className="flex justify-end gap-1.5">
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                aria-label={t('payrollScale.editDetail')}
                                onClick={() => {
                                  setEditing(row)
                                  setForm({
                                    kategoriStatusId: row.kategoriStatusId,
                                    tingkat: String(row.tingkat),
                                    nominal: String(row.nominal),
                                  })
                                  setFormError(null)
                                }}
                              >
                                <Pencil className="h-4 w-4" aria-hidden="true" />
                              </Button>
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                aria-label={t('payrollScale.deleteDetail')}
                                onClick={() => deleteMutation.mutate(row.id)}
                                disabled={deleteMutation.isPending}
                              >
                                <Trash2 className="h-4 w-4" aria-hidden="true" />
                              </Button>
                            </div>
                          </td>
                        ) : null}
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>

            {canManage ? (
              <form
                className="mt-4 grid gap-3 sm:grid-cols-[1fr_6.5rem_9rem_auto] sm:items-end"
                noValidate
                onSubmit={(event) => {
                  event.preventDefault()
                  saveMutation.mutate(form)
                }}
              >
                <div className="grid gap-1.5">
                  <Label htmlFor="kategori">{t('payrollScale.detailFormCategory')}</Label>
                  <Select
                    value={form.kategoriStatusId}
                    onValueChange={(value) =>
                      setForm((prev) => ({ ...prev, kategoriStatusId: value }))
                    }
                    disabled={kategori.isPending}
                  >
                    <SelectTrigger id="kategori" className="h-11">
                      <SelectValue placeholder="Pilih kategori" />
                    </SelectTrigger>
                    <SelectContent>
                      {(kategori.data ?? []).map((row) => (
                        <SelectItem key={row.id} value={row.id}>
                          {row.nama}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="tingkat">{t('payrollScale.detailFormLevel')}</Label>
                  <Input
                    id="tingkat"
                    type="number"
                    min={0}
                    max={99}
                    step={1}
                    required
                    value={form.tingkat}
                    onChange={(event) =>
                      setForm((prev) => ({ ...prev, tingkat: event.target.value }))
                    }
                    className="h-11"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="nominal">{t('payrollScale.detailFormNominal')}</Label>
                  <Input
                    id="nominal"
                    type="number"
                    min={0}
                    step="1"
                    required
                    value={form.nominal}
                    onChange={(event) =>
                      setForm((prev) => ({ ...prev, nominal: event.target.value }))
                    }
                    className="h-11"
                  />
                </div>
                <Button type="submit" disabled={saveMutation.isPending} className="h-11">
                  {editing ? (
                    <>
                      <Pencil className="mr-1.5 h-4 w-4" aria-hidden="true" />
                      {t('payrollScale.detailFormEdit')}
                    </>
                  ) : (
                    <>
                      <Plus className="mr-1.5 h-4 w-4" aria-hidden="true" />
                      {t('payrollScale.detailFormAdd')}
                    </>
                  )}
                </Button>
              </form>
            ) : null}
            {formError ? (
              <p role="alert" className="mt-2 text-sm text-destructive">
                {formError}
              </p>
            ) : null}
          </>
        )}

        <DialogFooter>
          {editing ? (
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setEditing(null)
                setForm(EMPTY_FORM)
                setFormError(null)
              }}
            >
              {t('payrollScale.cancel')}
            </Button>
          ) : null}
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t('payrollScale.cancel')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
