import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
} from '@tanstack/react-table'
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
  createHonorPeriode,
  deleteHonorPeriode,
  fetchPeriodeAkademik,
  honorSksQueryKey,
  updateHonorPeriode,
  useHonorPeriode,
} from '@/lib/honor-sks'
import {
  honorSksPeriodeCreateSchema,
  honorSksPeriodeUpdateSchema,
  type HonorSksPeriode,
} from '@/validations/schemas'

type HonorPeriodeSectionProps = {
  canManage: boolean
}

export default function HonorPeriodeSection({ canManage }: HonorPeriodeSectionProps) {
  const t = useMessages()
  const queryClient = useQueryClient()
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [editing, setEditing] = useState<HonorSksPeriode | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const periodeAkademik = useQuery({
    queryKey: ['periode-akademik'] as const,
    queryFn: fetchPeriodeAkademik,
    staleTime: 60_000,
  })

  const query = useHonorPeriode({ page, pageSize: 10, sort: 'userId', order: 'asc' })

  const saveMutation = useMutation({
    mutationFn: (values: {
      userId: string
      periodeAkademikId: string
      sks: number
      ratePerSks: number
    }) => {
      if (editing) {
        const parsed = honorSksPeriodeUpdateSchema.safeParse({
          sks: values.sks,
          ratePerSks: values.ratePerSks,
        })
        if (!parsed.success) {
          throw new Error(
            parsed.error.issues[0]?.message ?? t('honorSks.periodeSaveError'),
          )
        }
        return updateHonorPeriode(editing.id, parsed.data)
      }
      const parsed = honorSksPeriodeCreateSchema.safeParse(values)
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0]?.message ?? t('honorSks.periodeSaveError'))
      }
      return createHonorPeriode(parsed.data)
    },
    onSuccess: () => {
      toast.success(t('honorSks.periodeSaved'))
      setIsFormOpen(false)
      setEditing(null)
      setFormError(null)
      void queryClient.invalidateQueries({ queryKey: honorSksQueryKey })
    },
    onError: (error: Error) => {
      setFormError(error.message)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteHonorPeriode(id),
    onSuccess: () => {
      toast.success(t('honorSks.periodeDeleted'))
      void queryClient.invalidateQueries({ queryKey: honorSksQueryKey })
    },
    onError: () => {
      toast.error(t('honorSks.periodeDeleteError'))
    },
  })

  const columns: ColumnDef<HonorSksPeriode>[] = [
    {
      id: 'userId',
      header: () => t('honorSks.colUser'),
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.userId}</span>,
    },
    {
      id: 'periodeAkademikId',
      header: () => t('honorSks.colPeriode'),
      cell: ({ row }) => {
        const match = periodeAkademik.data?.find(
          (item) => item.id === row.original.periodeAkademikId,
        )
        return match ? (
          <span>
            {match.tahun} / Sem {match.semester}
          </span>
        ) : (
          <span className="font-mono text-xs">{row.original.periodeAkademikId}</span>
        )
      },
    },
    {
      id: 'sks',
      accessorKey: 'sks',
      header: () => t('honorSks.colSks'),
      cell: ({ row }) => <span className="tabular-nums">{row.original.sks}</span>,
    },
    {
      id: 'ratePerSks',
      accessorKey: 'ratePerSks',
      header: () => t('honorSks.colRate'),
      cell: ({ row }) => (
        <span className="tabular-nums">
          Rp {row.original.ratePerSks.toLocaleString('id-ID')}
        </span>
      ),
    },
    {
      id: 'totalHonor',
      accessorKey: 'totalHonor',
      header: () => t('honorSks.colTotal'),
      cell: ({ row }) => (
        <span className="font-semibold tabular-nums">
          Rp {row.original.totalHonor.toLocaleString('id-ID')}
        </span>
      ),
    },
    {
      id: 'actions',
      header: () => <span className="sr-only">{t('honorSks.colActions')}</span>,
      cell: ({ row }) =>
        canManage ? (
          <div className="flex justify-end gap-1.5">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              aria-label={t('honorSks.periodeEditTitle')}
              onClick={() => {
                setEditing(row.original)
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
              aria-label={t('honorSks.periodeTitle')}
              onClick={() => deleteMutation.mutate(row.original.id)}
              disabled={deleteMutation.isPending}
            >
              <Trash2 className="h-4 w-4" aria-hidden="true" />
            </Button>
          </div>
        ) : null,
    },
  ]

  const table = useReactTable({
    data: query.data?.items ?? [],
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  const periodeOptions = periodeAkademik.data ?? []

  return (
    <div className="py-2">
      <div className="mb-3 flex items-center justify-between gap-3">
        <div>
          <h2 className="m-0 text-lg font-semibold text-[var(--sea-ink)]">
            {t('honorSks.periodeTitle')}
          </h2>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('honorSks.periodeLead')}
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
            <Plus className="mr-1.5 h-4 w-4" aria-hidden="true" />
            {t('honorSks.addPeriode')}
          </Button>
        ) : null}
      </div>

      {query.isPending ? (
        <div className="py-8 text-center">
          <LoadingBars size="sm" label={t('honorSks.periodeTitle')} />
        </div>
      ) : query.isError || !query.data ? (
        <div className="py-8 text-center">
          <p className="mb-3 text-sm text-[var(--sea-ink-soft)]">
            {t('honorSks.periodeError')}
          </p>
          <RetryButton label={t('common.retry')} onClick={() => query.refetch()} />
        </div>
      ) : query.data.items.length === 0 ? (
        <p className="m-0 py-6 text-center text-sm text-[var(--sea-ink-soft)]">
          {t('honorSks.periodeEmpty')}
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-[var(--line)]">
          <table className="min-w-full border-collapse text-sm">
            <caption className="sr-only">{t('honorSks.periodeTitle')}</caption>
            <thead>
              <tr className="text-left text-xs uppercase tracking-[0.16em] text-[var(--kicker)]">
                {table.getHeaderGroups()[0]?.headers.map((header) => (
                  <th key={header.id} scope="col" className="px-3 py-2 font-semibold">
                    {flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {table.getRowModel().rows.map((row) => (
                <tr key={row.id} className="border-t border-[var(--line)]">
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id} className="px-3 py-2 align-middle">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {query.data && query.data.totalPages > 1 ? (
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
            <DialogTitle>
              {editing ? t('honorSks.periodeEditTitle') : t('honorSks.periodeFormTitle')}
            </DialogTitle>
            <DialogDescription>{t('honorSks.periodeLead')}</DialogDescription>
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
                periodeAkademikId: data.periodeAkademikId ?? '',
                sks: Number(data.sks),
                ratePerSks: Number(data.ratePerSks),
              })
            }}
          >
            <div className="grid gap-1.5">
              <Label htmlFor="userId">{t('honorSks.userId')}</Label>
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
              <Label htmlFor="periodeAkademikId">{t('honorSks.periodeField')}</Label>
              <Select
                name="periodeAkademikId"
                defaultValue={editing?.periodeAkademikId ?? ''}
                disabled={editing !== null}
                required
              >
                <SelectTrigger id="periodeAkademikId" className="h-11">
                  <SelectValue placeholder="Pilih periode" />
                </SelectTrigger>
                <SelectContent>
                  {periodeOptions.map((item) => (
                    <SelectItem key={item.id} value={item.id}>
                      {item.tahun} / Semester {item.semester}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="sks">{t('honorSks.sksLabel')}</Label>
                <Input
                  id="sks"
                  name="sks"
                  type="number"
                  min={0}
                  max={200}
                  step="0.5"
                  required
                  defaultValue={editing?.sks ?? ''}
                  className="h-11"
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="ratePerSks">{t('honorSks.ratePerSksLabel')}</Label>
                <Input
                  id="ratePerSks"
                  name="ratePerSks"
                  type="number"
                  min={0}
                  step="1000"
                  required
                  defaultValue={editing?.ratePerSks ?? ''}
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
