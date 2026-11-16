import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, ShieldAlert, ToggleLeft, ToggleRight } from 'lucide-react'
import { toast } from 'sonner'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
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
import LoadingBars from '@/components/LoadingBars'
import RetryButton from '@/components/RetryButton'
import PayrollScaleDetailDialog from '@/components/PayrollScaleDetailDialog'
import { useLocale, useMessages } from '@/i18n/locale-context'
import {
  PAYROLL_SCALE_SORT_FIELDS,
  createPayrollScale,
  updatePayrollScale,
  usePayrollScales,
  type PayrollScaleSortField,
} from '@/lib/payroll-scale'
import { useSession } from '@/lib/session'
import { useTableControls } from '@/lib/table-state'
import {
  payrollScaleCreateSchema,
  payrollScaleUpdateSchema,
  type PayrollScale,
} from '@/validations/schemas'

const SEARCH_DEBOUNCE_MS = 300

export default function PayrollScalePage() {
  const t = useMessages()
  const { locale } = useLocale()
  const session = useSession()
  const queryClient = useQueryClient()

  const permissions = session.data?.permissions ?? []
  const canRead =
    permissions.includes('payroll:read:any') ||
    permissions.includes('payroll:read:homebase') ||
    permissions.includes('payroll:manage')
  const canManage = permissions.includes('payroll:manage')

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [editing, setEditing] = useState<PayrollScale | null>(null)
  const [selected, setSelected] = useState<PayrollScale | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const controls = useTableControls<PayrollScaleSortField>({
    sortFields: PAYROLL_SCALE_SORT_FIELDS,
    initialSort: 'kode',
    initialPageSize: 25,
  })
  const { sort, order, page, pageSize, resetPage } = controls

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput.trim())
      resetPage()
    }, SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [searchInput, resetPage])

  const query = usePayrollScales({ page, pageSize, search, sort, order })

  const saveMutation = useMutation({
    mutationFn: async (values: {
      kode: string
      nama: string
      berlakuMulai: string
      berlakuSelesai: string | null
    }) => {
      const payload = {
        ...values,
        berlakuSelesai: values.berlakuSelesai === '' ? null : values.berlakuSelesai,
      }
      if (editing) {
        const parsed = payrollScaleUpdateSchema.safeParse(payload)
        if (!parsed.success) {
          throw new Error(parsed.error.issues[0]?.message ?? t('payrollScale.saveError'))
        }
        return updatePayrollScale(editing.id, parsed.data)
      }
      const parsed = payrollScaleCreateSchema.safeParse(payload)
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0]?.message ?? t('payrollScale.saveError'))
      }
      return createPayrollScale(parsed.data)
    },
    onSuccess: () => {
      toast.success(t('payrollScale.saved'))
      setIsFormOpen(false)
      setEditing(null)
      setFormError(null)
      void queryClient.invalidateQueries({ queryKey: ['payroll-scale'] })
    },
    onError: (error: Error) => {
      setFormError(error.message)
    },
  })

  const toggleDelete = useMutation({
    mutationFn: async (skala: PayrollScale) =>
      updatePayrollScale(skala.id, { aktif: !skala.aktif }),
    onSuccess: () => {
      toast.success(t('payrollScale.saved'))
      void queryClient.invalidateQueries({ queryKey: ['payroll-scale'] })
    },
    onError: () => {
      toast.error(t('payrollScale.saveError'))
    },
  })

  const columns = useMemo<ColumnDef<PayrollScale>[]>(
    () => [
      {
        id: 'kode',
        accessorKey: 'kode',
        header: () => t('payrollScale.colKode'),
        cell: ({ row }) => (
          <span className="font-mono text-sm font-semibold">{row.original.kode}</span>
        ),
      },
      {
        id: 'nama',
        accessorKey: 'nama',
        header: () => t('payrollScale.colNama'),
        cell: ({ row }) => <span className="font-medium">{row.original.nama}</span>,
      },
      {
        id: 'berlakuMulai',
        accessorKey: 'berlakuMulai',
        header: () => t('payrollScale.colPeriod'),
        cell: ({ row }) => {
          const start = new Date(row.original.berlakuMulai)
          const end = row.original.berlakuSelesai
            ? new Date(row.original.berlakuSelesai)
            : null
          const startLabel = new Intl.DateTimeFormat(
            locale === 'id' ? 'id-ID' : 'en-GB',
            { day: '2-digit', month: 'short', year: 'numeric' },
          ).format(start)
          const endLabel = end
            ? new Intl.DateTimeFormat(locale === 'id' ? 'id-ID' : 'en-GB', {
                day: '2-digit',
                month: 'short',
                year: 'numeric',
              }).format(end)
            : '∞'
          return (
            <span className="whitespace-nowrap text-sm text-[var(--sea-ink-soft)]">
              {startLabel} – {endLabel}
            </span>
          )
        },
      },
      {
        id: 'aktif',
        accessorKey: 'aktif',
        header: () => t('payrollScale.colAktif'),
        cell: ({ row }) => (
          <span
            className={
              row.original.aktif
                ? 'rounded-full bg-[var(--lagoon)] px-2 py-0.5 text-xs font-bold text-white'
                : 'rounded-full bg-[var(--sand)] px-2 py-0.5 text-xs font-semibold text-[var(--sea-ink-soft)]'
            }
          >
            {row.original.aktif ? 'Ya' : 'Tidak'}
          </span>
        ),
      },
      {
        id: 'updatedAt',
        accessorKey: 'updatedAt',
        header: () => t('payrollScale.colUpdatedAt'),
        cell: ({ row }) => (
          <span className="text-xs text-[var(--sea-ink-soft)]">
            {new Date(row.original.updatedAt).toLocaleString(
              locale === 'id' ? 'id-ID' : 'en-GB',
            )}
          </span>
        ),
      },
      {
        id: 'actions',
        header: () => <span className="sr-only">Aksi</span>,
        cell: ({ row }) => (
          <div className="flex justify-end gap-1.5">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-9 px-3"
              onClick={() => setSelected(row.original)}
            >
              {t('payrollScale.manageDetail')}
            </Button>
            {canManage ? (
              <>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-9 w-9"
                  aria-label={t('payrollScale.editTitle')}
                  onClick={() => {
                    setFormError(null)
                    setEditing(row.original)
                    setIsFormOpen(true)
                  }}
                >
                  <Pencil className="h-4 w-4" aria-hidden="true" />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-9 w-9"
                  aria-label={
                    row.original.aktif ? t('common.deactivate') : t('common.activate')
                  }
                  onClick={() => toggleDelete.mutate(row.original)}
                >
                  {row.original.aktif ? (
                    <ToggleRight className="h-4 w-4" aria-hidden="true" />
                  ) : (
                    <ToggleLeft className="h-4 w-4" aria-hidden="true" />
                  )}
                </Button>
              </>
            ) : null}
          </div>
        ),
      },
    ],
    [canManage, toggleDelete, t, locale],
  )

  const table = useReactTable({
    data: query.data?.items ?? [],
    columns,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    getRowId: (row) => row.id,
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

  const resultLabel = t('payrollScale.resultLabel', {
    count: new Intl.NumberFormat(locale === 'id' ? 'id-ID' : 'en-GB').format(
      query.data?.totalItems ?? 0,
    ),
  })

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('payrollScale.kicker')}</p>
      <div className="mb-2 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="display-title mb-2 text-3xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
            {t('payrollScale.title')}
          </h1>
          <p className="m-0 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)]">
            {t('payrollScale.lead')}
          </p>
        </div>
        {canManage ? (
          <Button
            type="button"
            className="gap-2 self-start sm:self-auto"
            onClick={() => {
              setFormError(null)
              setEditing(null)
              setIsFormOpen(true)
            }}
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            {t('payrollScale.addNew')}
          </Button>
        ) : null}
      </div>

      <div className="island-shell mt-4 rounded-2xl p-4 sm:p-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <label className="grid max-w-sm gap-1">
            <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('payrollScale.searchLabel')}
            </span>
            <Input
              type="search"
              value={searchInput}
              placeholder={t('payrollScale.searchPlaceholder')}
              onChange={(event) => setSearchInput(event.target.value)}
              className="h-11"
            />
          </label>
          <p className="m-0 text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
            {resultLabel}
          </p>
        </div>

        {query.isPending ? (
          <div className="py-10 text-center">
            <LoadingBars size="md" label={t('payrollScale.loadingTitle')} />
          </div>
        ) : query.isError || !query.data ? (
          <div className="py-10 text-center">
            <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
              {t('payrollScale.errorTitle')}
            </h2>
            <p className="m-0 mb-4 text-sm text-[var(--sea-ink-soft)]">
              {t('payrollScale.errorDescription')}
            </p>
            <RetryButton label={t('common.retry')} onClick={() => query.refetch()} />
          </div>
        ) : query.data.items.length === 0 ? (
          <p className="m-0 py-10 text-center text-sm text-[var(--sea-ink-soft)]">
            {t('payrollScale.empty')}
          </p>
        ) : (
          <div className="mt-3 overflow-x-auto">
            <table className="min-w-full border-collapse text-sm">
              <caption className="sr-only">{t('payrollScale.caption')}</caption>
              <thead>
                <tr className="border-b border-[var(--line)] text-left text-xs uppercase tracking-[0.16em] text-[var(--kicker)]">
                  {table.getFlatHeaders().map((header) => (
                    <th
                      key={header.id}
                      scope="col"
                      className="px-3 py-2 font-semibold"
                      aria-sort={undefined}
                    >
                      {flexRender(header.column.columnDef.header, header.getContext())}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {table.getRowModel().rows.map((row) => (
                  <tr
                    key={row.id}
                    className="border-b border-[var(--line)] last:border-0 align-middle text-[var(--sea-ink)]"
                  >
                    {row.getVisibleCells().map((cell) => (
                      <td key={cell.id} className="px-3 py-2.5 align-middle">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

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
              {editing ? t('payrollScale.editTitle') : t('payrollScale.createTitle')}
            </DialogTitle>
            <DialogDescription>{t('payrollScale.dialogDescription')}</DialogDescription>
          </DialogHeader>
          <form
            className="grid gap-4"
            noValidate
            onSubmit={(event) => {
              event.preventDefault()
              const form = event.currentTarget
              const data = Object.fromEntries(new FormData(form).entries()) as Record<
                string,
                string
              >
              const berlakuSelesai = (data.berlakuSelesai ?? '').trim()
              saveMutation.mutate({
                kode: data.kode?.trim() ?? '',
                nama: data.nama?.trim() ?? '',
                berlakuMulai: data.berlakuMulai?.trim() ?? '',
                berlakuSelesai: berlakuSelesai === '' ? null : berlakuSelesai,
              })
            }}
          >
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="kode">{t('payrollScale.fieldKode')}</Label>
                <Input
                  id="kode"
                  name="kode"
                  defaultValue={editing?.kode ?? ''}
                  required
                  maxLength={50}
                  className="h-11"
                  autoComplete="off"
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="nama">{t('payrollScale.fieldNama')}</Label>
                <Input
                  id="nama"
                  name="nama"
                  defaultValue={editing?.nama ?? ''}
                  required
                  maxLength={200}
                  className="h-11"
                  autoComplete="off"
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="berlakuMulai">{t('payrollScale.fieldStart')}</Label>
                <Input
                  id="berlakuMulai"
                  name="berlakuMulai"
                  type="date"
                  defaultValue={editing?.berlakuMulai?.slice(0, 10) ?? ''}
                  required
                  className="h-11"
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="berlakuSelesai">{t('payrollScale.fieldEnd')}</Label>
                <Input
                  id="berlakuSelesai"
                  name="berlakuSelesai"
                  type="date"
                  defaultValue={editing?.berlakuSelesai?.slice(0, 10) ?? ''}
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
                {t('payrollScale.cancel')}
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

      <PayrollScaleDetailDialog
        key={selected?.id ?? 'none'}
        scale={selected}
        open={selected !== null}
        onOpenChange={(open) => {
          if (!open) setSelected(null)
        }}
        canManage={canManage}
      />
    </div>
  )
}
