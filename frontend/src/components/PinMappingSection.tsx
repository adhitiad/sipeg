import { useRef, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Download, FileUp, Plus, Trash2 } from 'lucide-react'
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
  createMesinPin,
  deleteMesinPin,
  downloadMesinPinTemplate,
  importMesinPins,
  mesinPinQueryKey,
  useMesinPins,
  MAX_PIN_IMPORT_BYTES,
} from '@/lib/mesin-pin'
import type { PinImportResult } from '@/validations/schemas'

export default function PinMappingSection() {
  const t = useMessages()
  const queryClient = useQueryClient()
  const pins = useMesinPins()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [importIssues, setImportIssues] = useState<PinImportResult['issues'] | null>(null)

  const createMutation = useMutation({
    mutationFn: createMesinPin,
    onSuccess: () => {
      toast.success(t('mesinPin.created'))
      setIsFormOpen(false)
      setFormError(null)
      void queryClient.invalidateQueries({ queryKey: mesinPinQueryKey })
    },
    onError: (error: Error) => {
      setFormError(error.message)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteMesinPin(id),
    onSuccess: () => {
      toast.success(t('mesinPin.deleted'))
      void queryClient.invalidateQueries({ queryKey: mesinPinQueryKey })
    },
    onError: () => {
      toast.error(t('mesinPin.deleteError'))
    },
  })

  const importMutation = useMutation({
    mutationFn: (file: File) => importMesinPins(file),
    onSuccess: (result) => {
      setImportIssues(result.issues)
      if (result.issues.length === 0) {
        toast.success(t('mesinPin.importedAll', { count: String(result.imported) }))
      } else {
        toast.warning(
          t('mesinPin.importPartial', {
            imported: String(result.imported),
            issues: String(result.issues.length),
          }),
        )
      }
      void queryClient.invalidateQueries({ queryKey: mesinPinQueryKey })
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  if (pins.isPending) {
    return (
      <div className="py-8 text-center">
        <LoadingBars size="sm" label={t('mesinPin.title')} />
      </div>
    )
  }

  if (pins.isError || !pins.data) {
    return (
      <div className="py-8 text-center">
        <p className="mb-3 text-sm text-[var(--sea-ink-soft)]">
          {t('mesinPin.listError')}
        </p>
        <RetryButton label={t('common.retry')} onClick={() => pins.refetch()} />
      </div>
    )
  }

  const items = pins.data

  return (
    <div className="py-2">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="m-0 text-lg font-semibold text-[var(--sea-ink)]">
            {t('mesinPin.title')}
          </h2>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">{t('mesinPin.lead')}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void downloadMesinPinTemplate()}
          >
            <Download className="mr-1.5 h-4 w-4" aria-hidden="true" />
            {t('mesinPin.template')}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => fileInputRef.current?.click()}
            disabled={importMutation.isPending}
          >
            <FileUp className="mr-1.5 h-4 w-4" aria-hidden="true" />
            {importMutation.isPending ? t('mesinPin.importing') : t('mesinPin.importCsv')}
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept=".csv,.xlsx"
            className="sr-only"
            aria-label={t('mesinPin.importCsv')}
            onChange={(event) => {
              const file = event.target.files?.[0]
              if (!file) return
              if (file.size > MAX_PIN_IMPORT_BYTES) {
                toast.error(t('mesinPin.fileTooLarge'))
                return
              }
              importMutation.mutate(file)
              event.target.value = ''
            }}
          />
          <Button type="button" size="sm" onClick={() => setIsFormOpen(true)}>
            <Plus className="mr-1.5 h-4 w-4" aria-hidden="true" />
            {t('mesinPin.add')}
          </Button>
        </div>
      </div>

      {importIssues && importIssues.length > 0 ? (
        <div
          role="status"
          className="mb-3 rounded-xl border border-[var(--line)] bg-[var(--sand)] p-3 text-sm"
        >
          <p className="m-0 mb-2 font-semibold text-[var(--sea-ink)]">
            {t('mesinPin.issuesTitle', { count: String(importIssues.length) })}
          </p>
          <ul className="m-0 list-disc pl-5 text-[var(--sea-ink-soft)]">
            {importIssues.map((issue, index) => (
              <li key={index}>
                {t('mesinPin.issueRow', {
                  row: String(issue.row),
                  field: issue.field,
                  message: issue.message,
                })}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {items.length === 0 ? (
        <p className="m-0 py-6 text-center text-sm text-[var(--sea-ink-soft)]">
          {t('mesinPin.empty')}
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-[var(--line)]">
          <table className="min-w-full border-collapse text-sm">
            <caption className="sr-only">{t('mesinPin.title')}</caption>
            <thead>
              <tr className="text-left text-xs uppercase tracking-[0.16em] text-[var(--kicker)]">
                <th scope="col" className="px-3 py-2">
                  {t('mesinPin.colPin')}
                </th>
                <th scope="col" className="px-3 py-2">
                  {t('mesinPin.colNama')}
                </th>
                <th scope="col" className="px-3 py-2">
                  {t('mesinPin.colCatatan')}
                </th>
                <th scope="col" className="px-3 py-2 text-right">
                  <span className="sr-only">Aksi</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {items.map((row) => (
                <tr key={row.id} className="border-t border-[var(--line)] align-middle">
                  <td className="px-3 py-2 font-mono text-xs font-semibold">
                    {row.machinePin}
                  </td>
                  <td className="px-3 py-2">{row.nama}</td>
                  <td className="px-3 py-2 text-[var(--sea-ink-soft)]">
                    {row.catatan ?? '—'}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      aria-label={t('mesinPin.deleteAria', { pin: row.machinePin })}
                      onClick={() => deleteMutation.mutate(row.id)}
                      disabled={deleteMutation.isPending}
                    >
                      <Trash2 className="h-4 w-4" aria-hidden="true" />
                    </Button>
                  </td>
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
          if (!open) setFormError(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('mesinPin.add')}</DialogTitle>
            <DialogDescription>{t('mesinPin.dialogLead')}</DialogDescription>
          </DialogHeader>
          <form
            className="grid gap-4"
            noValidate
            onSubmit={(event) => {
              event.preventDefault()
              const data = Object.fromEntries(
                new FormData(event.currentTarget).entries(),
              ) as Record<string, string>
              const catatan = data.catatan?.trim()
              createMutation.mutate({
                machinePin: data.machinePin?.trim() ?? '',
                userId: data.userId?.trim() ?? '',
                catatan: catatan ? catatan : null,
              })
            }}
          >
            <div className="grid gap-1.5">
              <Label htmlFor="machinePin">{t('mesinPin.colPin')}</Label>
              <Input
                id="machinePin"
                name="machinePin"
                required
                className="h-11 font-mono"
                autoComplete="off"
                placeholder="1001"
              />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="userId">{t('mesinPin.userId')}</Label>
              <Input
                id="userId"
                name="userId"
                required
                className="h-11 font-mono text-xs"
                autoComplete="off"
                placeholder="00000000-0000-4000-8000-..."
              />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="catatan">{t('mesinPin.colCatatan')}</Label>
              <Input
                id="catatan"
                name="catatan"
                maxLength={200}
                className="h-11"
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
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending
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
