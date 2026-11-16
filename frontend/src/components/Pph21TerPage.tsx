import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Trash2, ShieldAlert } from 'lucide-react'
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
  createPph21Ter,
  deletePph21Ter,
  pph21TerQueryKey,
  updatePph21Ter,
  usePph21Ter,
} from '@/lib/pph21-ter'
import { pph21TerTarifCreateSchema, type Pph21TerTarif } from '@/validations/schemas'
import { useSession } from '@/lib/session'

export default function Pph21TerPage() {
  const t = useMessages()
  const session = useSession()
  const queryClient = useQueryClient()
  const permissions = session.data?.permissions ?? []
  const canRead =
    permissions.includes('payroll:read:any') ||
    permissions.includes('payroll:read:homebase') ||
    permissions.includes('payroll:manage')
  const canManage = permissions.includes('payroll:manage')

  const rates = usePph21Ter()
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [editing, setEditing] = useState<Pph21TerTarif | null>(null)
  const [formError, setFormError] = useState<string | null>(null)

  const saveMutation = useMutation({
    mutationFn: (values: {
      kategori: 'A' | 'B' | 'C'
      lapisan: number
      batasBawah: number
      batasAtas: number | null
      tarifPct: number
      berlakuMulai: string
    }) => {
      const parsed = pph21TerTarifCreateSchema.safeParse({
        ...values,
        batasAtas: values.batasAtas ?? undefined,
      })
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0]?.message ?? t('pph21.saveError'))
      }
      if (editing) {
        return updatePph21Ter(editing.id, parsed.data)
      }
      return createPph21Ter(parsed.data)
    },
    onSuccess: () => {
      toast.success(t('pph21.saved'))
      setIsFormOpen(false)
      setEditing(null)
      setFormError(null)
      void queryClient.invalidateQueries({ queryKey: pph21TerQueryKey })
    },
    onError: (error: Error) => {
      setFormError(error.message)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deletePph21Ter(id),
    onSuccess: () => {
      toast.success(t('pph21.deleted'))
      void queryClient.invalidateQueries({ queryKey: pph21TerQueryKey })
    },
    onError: () => {
      toast.error(t('pph21.saveError'))
    },
  })

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

  if (rates.isPending) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <LoadingBars size="md" label={t('pph21.title')} />
        </section>
      </div>
    )
  }

  if (rates.isError || !rates.data) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <p className="mb-3 text-sm text-[var(--sea-ink-soft)]">{t('pph21.error')}</p>
          <RetryButton label={t('common.retry')} onClick={() => rates.refetch()} />
        </section>
      </div>
    )
  }

  const items = rates.data
  const grouped = ['A', 'B', 'C'] as const

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('pph21.kicker')}</p>
      <div className="mb-2 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="display-title mb-2 text-3xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
            {t('pph21.title')}
          </h1>
          <p className="m-0 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)]">
            {t('pph21.lead')}
          </p>
        </div>
        {canManage ? (
          <Button
            type="button"
            className="gap-2 self-start sm:self-auto"
            onClick={() => {
              setEditing(null)
              setFormError(null)
              setIsFormOpen(true)
            }}
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            {t('pph21.add')}
          </Button>
        ) : null}
      </div>

      {items.length === 0 ? (
        <p className="m-0 mt-6 text-center text-sm text-[var(--sea-ink-soft)]">
          {t('pph21.empty')}
        </p>
      ) : (
        grouped.map((kategori) => {
          const rows = items.filter((row) => row.kategori === kategori)
          if (rows.length === 0) return null
          return (
            <section
              key={kategori}
              className="island-shell mt-4 overflow-hidden rounded-2xl"
            >
              <h2 className="m-0 border-b border-[var(--line)] bg-[var(--sand)] px-4 py-3 text-base font-bold tracking-[0.12em] text-[var(--sea-ink)]">
                Kategori {kategori}
              </h2>
              <div className="overflow-x-auto">
                <table className="min-w-full border-collapse text-sm">
                  <caption className="sr-only">PPh 21 TER kategori {kategori}</caption>
                  <thead>
                    <tr className="text-left text-xs uppercase tracking-[0.16em] text-[var(--kicker)]">
                      <th scope="col" className="px-3 py-2">
                        {t('pph21.colLapisan')}
                      </th>
                      <th scope="col" className="px-3 py-2 text-right">
                        {t('pph21.colBatasBawah')}
                      </th>
                      <th scope="col" className="px-3 py-2 text-right">
                        {t('pph21.colBatasAtas')}
                      </th>
                      <th scope="col" className="px-3 py-2 text-right">
                        {t('pph21.colTarif')}
                      </th>
                      <th scope="col" className="px-3 py-2">
                        {t('pph21.colBerlaku')}
                      </th>
                      {canManage ? (
                        <th scope="col" className="px-3 py-2 text-right">
                          <span className="sr-only">Aksi</span>
                        </th>
                      ) : null}
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((row) => (
                      <tr
                        key={row.id}
                        className="border-t border-[var(--line)] align-middle"
                      >
                        <td className="px-3 py-2 tabular-nums">{row.lapisan}</td>
                        <td className="px-3 py-2 text-right tabular-nums">
                          Rp {row.batasBawah.toLocaleString('id-ID')}
                        </td>
                        <td className="px-3 py-2 text-right tabular-nums">
                          {row.batasAtas != null
                            ? `Rp ${row.batasAtas.toLocaleString('id-ID')}`
                            : '∞'}
                        </td>
                        <td className="px-3 py-2 text-right tabular-nums">
                          {row.tarifPct.toLocaleString('id-ID')}%
                        </td>
                        <td className="px-3 py-2">
                          {new Date(row.berlakuMulai).toLocaleDateString('id-ID', {
                            day: '2-digit',
                            month: 'short',
                            year: 'numeric',
                          })}
                        </td>
                        {canManage ? (
                          <td className="px-3 py-2 text-right">
                            <div className="flex justify-end gap-1.5">
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                aria-label={t('pph21.dialogEdit')}
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
                                size="icon"
                                className="h-8 w-8"
                                aria-label={t('pph21.deleted')}
                                onClick={() => deleteMutation.mutate(row.id)}
                                disabled={deleteMutation.isPending}
                              >
                                <Trash2 className="h-4 w-4" aria-hidden="true" />
                              </Button>
                            </div>
                          </td>
                        ) : null}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          )
        })
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
              {editing ? t('pph21.dialogEdit') : t('pph21.dialogCreate')}
            </DialogTitle>
            <DialogDescription>{t('pph21.lead')}</DialogDescription>
          </DialogHeader>
          <form
            className="grid gap-4"
            noValidate
            onSubmit={(event) => {
              event.preventDefault()
              const data = Object.fromEntries(
                new FormData(event.currentTarget).entries(),
              ) as Record<string, string>
              const batasAtas = data.batasAtas?.trim() ?? ''
              saveMutation.mutate({
                kategori: (data.kategori as 'A' | 'B' | 'C') ?? 'A',
                lapisan: Number(data.lapisan),
                batasBawah: Number(data.batasBawah),
                batasAtas: batasAtas === '' ? null : Number(batasAtas),
                tarifPct: Number(data.tarifPct),
                berlakuMulai: data.berlakuMulai?.trim() ?? '',
              })
            }}
          >
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="kategori">{t('pph21.fieldKategori')}</Label>
                <Select name="kategori" defaultValue={editing?.kategori ?? 'A'}>
                  <SelectTrigger id="kategori" className="h-11">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="A">A</SelectItem>
                    <SelectItem value="B">B</SelectItem>
                    <SelectItem value="C">C</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="lapisan">{t('pph21.fieldLapisan')}</Label>
                <Input
                  id="lapisan"
                  name="lapisan"
                  type="number"
                  min={1}
                  step="1"
                  required
                  defaultValue={editing?.lapisan ?? ''}
                  className="h-11"
                />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="batasBawah">{t('pph21.fieldBatasBawah')}</Label>
                <Input
                  id="batasBawah"
                  name="batasBawah"
                  type="number"
                  min={0}
                  step="1"
                  required
                  defaultValue={editing?.batasBawah ?? ''}
                  className="h-11"
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="batasAtas">{t('pph21.fieldBatasAtas')}</Label>
                <Input
                  id="batasAtas"
                  name="batasAtas"
                  type="number"
                  min={0}
                  step="1"
                  defaultValue={editing?.batasAtas ?? ''}
                  className="h-11"
                />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="tarifPct">{t('pph21.fieldTarif')}</Label>
                <Input
                  id="tarifPct"
                  name="tarifPct"
                  type="number"
                  min={0}
                  max={100}
                  step="0.01"
                  required
                  defaultValue={editing?.tarifPct ?? ''}
                  className="h-11"
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="berlakuMulai">{t('pph21.fieldBerlaku')}</Label>
                <Input
                  id="berlakuMulai"
                  name="berlakuMulai"
                  type="date"
                  required
                  defaultValue={editing?.berlakuMulai?.slice(0, 10) ?? ''}
                  disabled={editing !== null}
                  className="h-11"
                />
              </div>
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
