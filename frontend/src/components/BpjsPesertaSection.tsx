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
  createBpjsPeserta,
  updateBpjsPeserta,
  useBpjsPeserta,
} from '@/lib/bpjs'
import {
  bpjsPesertaCreateSchema,
  bpjsPesertaUpdateSchema,
  type BpjsPeserta,
} from '@/validations/schemas'

type BpjsPesertaSectionProps = {
  canManage: boolean
}

export default function BpjsPesertaSection({ canManage }: BpjsPesertaSectionProps) {
  const t = useMessages()
  const queryClient = useQueryClient()
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [editing, setEditing] = useState<BpjsPeserta | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [page, setPage] = useState(1)

  const query = useBpjsPeserta({ page, pageSize: 10, sort: 'userId', order: 'asc' })

  const saveMutation = useMutation({
    mutationFn: (values: {
      userId: string
      jenis: 'KESEHATAN' | 'KETENAGAKERJAAN'
      nomorKepesertaan: string
      mulai: string
      selesai: string
    }) => {
      const payload = {
        nomorKepesertaan: values.nomorKepesertaan === '' ? null : values.nomorKepesertaan,
        mulai: values.mulai,
        selesai: values.selesai === '' ? null : values.selesai,
        aktif: true,
      }
      if (editing) {
        const parsed = bpjsPesertaUpdateSchema.safeParse({
          nomorKepesertaan: payload.nomorKepesertaan,
          mulai: payload.mulai,
          selesai: payload.selesai,
        })
        if (!parsed.success) {
          throw new Error(parsed.error.issues[0]?.message ?? t('bpjs.pesertaSaveError'))
        }
        return updateBpjsPeserta(editing.id, parsed.data)
      }
      const parsed = bpjsPesertaCreateSchema.safeParse({
        userId: values.userId,
        jenis: values.jenis,
        nomorKepesertaan: values.nomorKepesertaan === '' ? null : values.nomorKepesertaan,
        mulai: values.mulai,
        selesai: payload.selesai,
      })
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0]?.message ?? t('bpjs.pesertaSaveError'))
      }
      return createBpjsPeserta(parsed.data)
    },
    onSuccess: () => {
      toast.success(t('bpjs.pesertaSaved'))
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
    mutationFn: (row: BpjsPeserta) => updateBpjsPeserta(row.id, { aktif: !row.aktif }),
    onSuccess: () => {
      toast.success(t('bpjs.pesertaSaved'))
      void queryClient.invalidateQueries({ queryKey: bpjsQueryKey })
    },
    onError: () => {
      toast.error(t('bpjs.pesertaSaveError'))
    },
  })

  if (query.isPending) {
    return (
      <div className="py-8 text-center">
        <LoadingBars size="sm" label={t('bpjs.pesertaTitle')} />
      </div>
    )
  }

  if (query.isError || !query.data) {
    return (
      <div className="py-8 text-center">
        <p className="mb-3 text-sm text-[var(--sea-ink-soft)]">
          {t('bpjs.pesertaError')}
        </p>
        <RetryButton label={t('common.retry')} onClick={() => query.refetch()} />
      </div>
    )
  }

  const items = query.data.items

  return (
    <div className="py-2">
      <div className="mb-3 flex items-center justify-between gap-3">
        <div>
          <h2 className="m-0 text-lg font-semibold text-[var(--sea-ink)]">
            {t('bpjs.pesertaTitle')}
          </h2>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('bpjs.pesertaLead')}
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
            {t('bpjs.addPeserta')}
          </Button>
        ) : null}
      </div>

      {items.length === 0 ? (
        <p className="m-0 py-6 text-center text-sm text-[var(--sea-ink-soft)]">
          {t('bpjs.pesertaEmpty')}
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-[var(--line)]">
          <table className="min-w-full border-collapse text-sm">
            <caption className="sr-only">{t('bpjs.pesertaTitle')}</caption>
            <thead>
              <tr className="text-left text-xs uppercase tracking-[0.16em] text-[var(--kicker)]">
                <th scope="col" className="px-3 py-2">
                  {t('honorSks.colUser')}
                </th>
                <th scope="col" className="px-3 py-2">
                  {t('bpjs.colJenis')}
                </th>
                <th scope="col" className="px-3 py-2">
                  {t('bpjs.nomorPeserta')}
                </th>
                <th scope="col" className="px-3 py-2">
                  {t('bpjs.mulai')}
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
                  <td className="px-3 py-2 font-mono text-xs">{row.userId}</td>
                  <td className="px-3 py-2">{row.jenis}</td>
                  <td className="px-3 py-2">{row.nomorKepesertaan ?? '—'}</td>
                  <td className="px-3 py-2">
                    {new Date(row.mulai).toLocaleDateString('id-ID', {
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
                          aria-label={t('bpjs.pesertaDialogTitle')}
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

      {query.data.totalPages > 1 ? (
        <div className="mt-3 flex items-center justify-between text-xs text-[var(--sea-ink-soft)]">
          <span>
            Halaman {query.data.page} dari {query.data.totalPages}
          </span>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
            >
              {t('common.prevPage')}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={page >= query.data.totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              {t('common.nextPage')}
            </Button>
          </div>
        </div>
      ) : null}

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
            <DialogTitle>{t('bpjs.pesertaDialogTitle')}</DialogTitle>
            <DialogDescription>{t('bpjs.pesertaLead')}</DialogDescription>
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
                userId: data.userId?.trim() ?? '',
                jenis: (data.jenis as 'KESEHATAN' | 'KETENAGAKERJAAN') ?? 'KESEHATAN',
                nomorKepesertaan: data.nomorKepesertaan?.trim() ?? '',
                mulai: data.mulai?.trim() ?? '',
                selesai: data.selesai?.trim() ?? '',
              })
            }}
          >
            <div className="grid gap-1.5">
              <Label htmlFor="userId">{t('bpjs.userId')}</Label>
              <Input
                id="userId"
                name="userId"
                placeholder="00000000-0000-4000-8000-..."
                required
                defaultValue={editing?.userId ?? ''}
                disabled={editing !== null}
                className="h-11 font-mono text-xs"
                autoComplete="off"
              />
            </div>
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
                <Label htmlFor="mulai">{t('bpjs.mulai')}</Label>
                <Input
                  id="mulai"
                  name="mulai"
                  type="date"
                  required
                  defaultValue={editing?.mulai?.slice(0, 10) ?? ''}
                  className="h-11"
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="selesai">{t('bpjs.selesai')}</Label>
                <Input
                  id="selesai"
                  name="selesai"
                  type="date"
                  defaultValue={editing?.selesai?.slice(0, 10) ?? ''}
                  className="h-11"
                />
              </div>
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="nomorKepesertaan">{t('bpjs.nomorPeserta')}</Label>
              <Input
                id="nomorKepesertaan"
                name="nomorKepesertaan"
                placeholder="TKT-000123"
                defaultValue={editing?.nomorKepesertaan ?? ''}
                className="h-11 font-mono text-xs"
                autoComplete="off"
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
