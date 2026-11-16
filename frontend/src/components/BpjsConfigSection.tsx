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
  bpjsQueryKey,
  createBpjsKonfigurasi,
  updateBpjsKonfigurasi,
  useBpjsKonfigurasi,
} from '@/lib/bpjs'
import { bpjsKonfigurasiCreateSchema, type BpjsKonfigurasi } from '@/validations/schemas'

type BpjsConfigSectionProps = {
  canManage: boolean
}

export default function BpjsConfigSection({ canManage }: BpjsConfigSectionProps) {
  const t = useMessages()
  const queryClient = useQueryClient()
  const config = useBpjsKonfigurasi()
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [editing, setEditing] = useState<BpjsKonfigurasi | null>(null)
  const [formError, setFormError] = useState<string | null>(null)

  const saveMutation = useMutation({
    mutationFn: (values: {
      jenis: 'KESEHATAN' | 'KETENAGAKERJAAN'
      iuranPegawaiPct: number
      iuranPemberiKerjaPct: number
      dasar: 'GAJI_POKOK' | 'TOTAL_GAJI'
      batasAtas: string
      berlakuMulai: string
    }) => {
      const payload = {
        ...values,
        batasAtas: values.batasAtas === '' ? null : Number(values.batasAtas),
      }
      if (editing) {
        return updateBpjsKonfigurasi(editing.id, {
          iuranPegawaiPct: payload.iuranPegawaiPct,
          iuranPemberiKerjaPct: payload.iuranPemberiKerjaPct,
          dasar: payload.dasar,
          batasAtas: payload.batasAtas,
        })
      }
      const parsed = bpjsKonfigurasiCreateSchema.safeParse(payload)
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0]?.message ?? t('bpjs.configSaveError'))
      }
      return createBpjsKonfigurasi(parsed.data)
    },
    onSuccess: () => {
      toast.success(t('bpjs.configSaved'))
      setIsFormOpen(false)
      setEditing(null)
      setFormError(null)
      void queryClient.invalidateQueries({ queryKey: bpjsQueryKey })
    },
    onError: (error: Error) => {
      setFormError(error.message)
    },
  })

  const toggleMutation = useMutation({
    mutationFn: (row: BpjsKonfigurasi) =>
      updateBpjsKonfigurasi(row.id, { aktif: !row.aktif }),
    onSuccess: () => {
      toast.success(t('bpjs.configSaved'))
      void queryClient.invalidateQueries({ queryKey: bpjsQueryKey })
    },
    onError: () => {
      toast.error(t('bpjs.configSaveError'))
    },
  })

  if (config.isPending) {
    return (
      <div className="py-8 text-center">
        <LoadingBars size="sm" label={t('bpjs.configTitle')} />
      </div>
    )
  }

  if (config.isError || !config.data) {
    return (
      <div className="py-8 text-center">
        <p className="mb-3 text-sm text-[var(--sea-ink-soft)]">{t('bpjs.configError')}</p>
        <RetryButton label={t('common.retry')} onClick={() => config.refetch()} />
      </div>
    )
  }

  const items = config.data

  return (
    <div className="py-2">
      <div className="mb-3 flex items-center justify-between gap-3">
        <div>
          <h2 className="m-0 text-lg font-semibold text-[var(--sea-ink)]">
            {t('bpjs.configTitle')}
          </h2>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">{t('bpjs.configLead')}</p>
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
            {t('bpjs.addConfig')}
          </Button>
        ) : null}
      </div>

      {items.length === 0 ? (
        <p className="m-0 py-6 text-center text-sm text-[var(--sea-ink-soft)]">
          {t('bpjs.configEmpty')}
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-[var(--line)]">
          <table className="min-w-full border-collapse text-sm">
            <caption className="sr-only">{t('bpjs.configTitle')}</caption>
            <thead>
              <tr className="text-left text-xs uppercase tracking-[0.16em] text-[var(--kicker)]">
                <th scope="col" className="px-3 py-2">
                  {t('bpjs.colJenis')}
                </th>
                <th scope="col" className="px-3 py-2 text-right">
                  {t('bpjs.colPegawai')}
                </th>
                <th scope="col" className="px-3 py-2 text-right">
                  {t('bpjs.colPemberiKerja')}
                </th>
                <th scope="col" className="px-3 py-2">
                  {t('bpjs.colDasar')}
                </th>
                <th scope="col" className="px-3 py-2 text-right">
                  {t('bpjs.colBatasAtas')}
                </th>
                <th scope="col" className="px-3 py-2">
                  {t('bpjs.colBerlaku')}
                </th>
                <th scope="col" className="px-3 py-2">
                  {t('bpjs.colAktif')}
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
                  <td className="px-3 py-2 font-medium">{row.jenis}</td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {row.iuranPegawaiPct}%
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {row.iuranPemberiKerjaPct}%
                  </td>
                  <td className="px-3 py-2">{row.dasar}</td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {row.batasAtas != null
                      ? `Rp ${row.batasAtas.toLocaleString('id-ID')}`
                      : '—'}
                  </td>
                  <td className="px-3 py-2">
                    {new Date(row.berlakuMulai).toLocaleDateString('id-ID', {
                      day: '2-digit',
                      month: 'short',
                      year: 'numeric',
                    })}
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
                          aria-label={t('bpjs.configEditTitle')}
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
              {editing ? t('bpjs.configEditTitle') : t('bpjs.configDialogTitle')}
            </DialogTitle>
            <DialogDescription>{t('bpjs.configLead')}</DialogDescription>
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
                jenis: (data.jenis as 'KESEHATAN' | 'KETENAGAKERJAAN') ?? 'KESEHATAN',
                iuranPegawaiPct: Number(data.iuranPegawaiPct),
                iuranPemberiKerjaPct: Number(data.iuranPemberiKerjaPct),
                dasar: (data.dasar as 'GAJI_POKOK' | 'TOTAL_GAJI') ?? 'GAJI_POKOK',
                batasAtas: data.batasAtas?.trim() ?? '',
                berlakuMulai: data.berlakuMulai?.trim() ?? '',
              })
            }}
          >
            <div className="grid gap-1.5">
              <Label htmlFor="jenis">{t('bpjs.fieldJenis')}</Label>
              <Select
                name="jenis"
                defaultValue={editing?.jenis ?? 'KESEHATAN'}
                disabled={editing !== null}
              >
                <SelectTrigger id="jenis" className="h-11">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="KESEHATAN">KESEHATAN</SelectItem>
                  <SelectItem value="KETENAGAKERJAAN">KETENAGAKERJAAN</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="iuranPegawaiPct">{t('bpjs.fieldPegawaiPct')}</Label>
                <Input
                  id="iuranPegawaiPct"
                  name="iuranPegawaiPct"
                  type="number"
                  min={0}
                  max={100}
                  step="0.01"
                  required
                  defaultValue={editing?.iuranPegawaiPct ?? ''}
                  disabled={editing !== null}
                  className="h-11"
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="iuranPemberiKerjaPct">
                  {t('bpjs.fieldPemberiKerjaPct')}
                </Label>
                <Input
                  id="iuranPemberiKerjaPct"
                  name="iuranPemberiKerjaPct"
                  type="number"
                  min={0}
                  max={100}
                  step="0.01"
                  required
                  defaultValue={editing?.iuranPemberiKerjaPct ?? ''}
                  className="h-11"
                />
              </div>
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="dasar">{t('bpjs.fieldDasar')}</Label>
              <Select name="dasar" defaultValue={editing?.dasar ?? 'GAJI_POKOK'}>
                <SelectTrigger id="dasar" className="h-11">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="GAJI_POKOK">GAJI_POKOK</SelectItem>
                  <SelectItem value="TOTAL_GAJI">TOTAL_GAJI</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="batasAtas">{t('bpjs.fieldBatasAtas')}</Label>
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
              <div className="grid gap-1.5">
                <Label htmlFor="berlakuMulai">{t('bpjs.fieldBerlaku')}</Label>
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
