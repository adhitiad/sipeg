import { useRef, useState } from 'react'
import { Download, Upload } from 'lucide-react'
import { toast } from 'sonner'
import { ApiError } from '@/lib/api'
import {
  downloadEmployeeImportTemplate,
  employeeImportFileTooLarge,
  importEmployees,
} from '@/lib/employee-import'
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
import { useMessages } from '@/i18n/locale-context'

type EmployeeImportDialogProps = {
  open: boolean
  onClose: () => void
  onImported?: () => void
}

type ImportIssue = { row: number; field?: string; code?: string; message?: string }

function parseImportIssues(rows: unknown): ImportIssue[] {
  if (!Array.isArray(rows)) {
    return []
  }
  return rows.flatMap((item) => {
    if (typeof item !== 'object' || item === null) {
      return []
    }
    const record = item as Record<string, unknown>
    const row = Number(record.row)
    if (!Number.isFinite(row)) {
      return []
    }
    return [
      {
        row,
        field: typeof record.field === 'string' ? record.field : undefined,
        code: typeof record.code === 'string' ? record.code : undefined,
        message: typeof record.message === 'string' ? record.message : undefined,
      },
    ]
  })
}

export default function EmployeeImportDialog({
  open,
  onClose,
  onImported,
}: EmployeeImportDialogProps) {
  const t = useMessages()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [issues, setIssues] = useState<ImportIssue[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)

  function reset() {
    setFile(null)
    setIssues([])
    setIsSubmitting(false)
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  function handleClose() {
    reset()
    onClose()
  }

  function handleSubmit() {
    if (!file) {
      return
    }
    if (employeeImportFileTooLarge(file)) {
      toast.error(t('employees.importTooLarge'))
      return
    }
    setIsSubmitting(true)
    setIssues([])
    importEmployees(file)
      .then((result) => {
        onImported?.()
        toast.success(t('employees.importSuccess', { count: String(result.imported) }))
        handleClose()
      })
      .catch((error: unknown) => {
        setIsSubmitting(false)
        if (error instanceof ApiError && error.code === 'validation_error') {
          setIssues(parseImportIssues(error.rows))
          toast.error(t('employees.importRowErrors'))
          return
        }
        if (error instanceof ApiError && error.code === 'identity_conflict') {
          toast.error(t('employees.importConflict'))
          return
        }
        if (error instanceof ApiError && error.status === 413) {
          toast.error(t('employees.importTooLarge'))
          return
        }
        toast.error(t('employees.importError'))
      })
  }

  return (
    <Dialog open={open} onOpenChange={(next) => !next && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('employees.importTitle')}</DialogTitle>
          <DialogDescription>{t('employees.importDescription')}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-3">
          <Button
            type="button"
            variant="outline"
            onClick={() => void downloadEmployeeImportTemplate()}
          >
            <Download className="h-4 w-4" aria-hidden="true" />
            {t('employees.importTemplate')}
          </Button>
          <div>
            <Label htmlFor="import-file">{t('employees.importFileLabel')}</Label>
            <Input
              id="import-file"
              ref={fileInputRef}
              type="file"
              accept=".csv,.xls,.xlsx"
              onChange={(event) => {
                setFile(event.target.files?.[0] ?? null)
                setIssues([])
              }}
              className="h-11"
            />
            <p className="mt-1 text-xs text-[var(--sea-ink-soft)]">
              {t('employees.importMaxNote')}
            </p>
          </div>
          {issues.length > 0 ? (
            <div className="max-h-48 overflow-y-auto rounded-xl border border-[var(--line)] p-3">
              <p className="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
                {t('employees.importErrorsTitle')}
              </p>
              <ul className="m-0 list-none space-y-1 p-0 text-xs text-[var(--sea-ink-soft)]">
                {issues.map((issue, index) => (
                  <li key={`${issue.row}-${index}`}>
                    {t('employees.importIssueRow', {
                      row: String(issue.row),
                      detail: [issue.field, issue.message].filter(Boolean).join(': '),
                    })}
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            {t('common.cancel')}
          </Button>
          <Button type="button" onClick={handleSubmit} disabled={!file || isSubmitting}>
            <Upload className="h-4 w-4" aria-hidden="true" />
            {t('employees.importSubmit')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
