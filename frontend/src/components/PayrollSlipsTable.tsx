import { Link } from '@tanstack/react-router'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import SortIcon from '@/components/SortIcon'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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
import type { PayrollSlip } from '@/validations/schemas'

const PAGE_SIZES = [10, 25, 50, 100] as const
export type PayrollPageSize = (typeof PAGE_SIZES)[number]

type PayrollSlipsTableProps = {
  slips: PayrollSlip[]
  page: number
  pageSize: PayrollPageSize
  totalPages: number
  totalItems: number
  isFetching: boolean
  search: string
  status: string | null
  sort: 'periode' | 'status' | 'createdAt' | 'updatedAt'
  order: 'asc' | 'desc'
  onPageChange: (page: number) => void
  onPageSizeChange: (size: PayrollPageSize) => void
  onSearchChange: (value: string) => void
  onStatusChange: (value: string | null) => void
  onSortChange: (field: 'periode' | 'status' | 'createdAt' | 'updatedAt') => void
}

export default function PayrollSlipsTable({
  slips,
  page,
  pageSize,
  totalPages,
  totalItems,
  isFetching,
  search,
  status,
  sort,
  order,
  onPageChange,
  onPageSizeChange,
  onSearchChange,
  onStatusChange,
  onSortChange,
}: PayrollSlipsTableProps) {
  const t = useMessages()
  const safeTotalPages = Math.max(1, totalPages)
  const canPrev = page > 1
  const canNext = page < safeTotalPages

  return (
    <div>
      <div className="mb-4 grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto_auto] sm:items-end">
        <label className="grid gap-1">
          <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
            {t('payroll.searchLabel')}
          </span>
          <Input
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder={t('payroll.searchPlaceholder')}
            className="h-11"
          />
        </label>
        <label className="grid gap-1">
          <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
            {t('payroll.filterStatus')}
          </span>
          <Select
            value={status ?? 'ALL'}
            onValueChange={(value) => onStatusChange(value === 'ALL' ? null : value)}
          >
            <SelectTrigger className="h-11 w-[180px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">{t('payroll.allStatuses')}</SelectItem>
              <SelectItem value="DRAF">{t('payroll.statusDraft')}</SelectItem>
              <SelectItem value="TERBIT">{t('payroll.statusIssued')}</SelectItem>
              <SelectItem value="TERBAYAR">{t('payroll.statusPaid')}</SelectItem>
            </SelectContent>
          </Select>
        </label>
        <label className="grid gap-1">
          <span className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
            {t('payroll.pageSizeLabel')}
          </span>
          <Select
            value={String(pageSize)}
            onValueChange={(value) => onPageSizeChange(Number(value) as PayrollPageSize)}
          >
            <SelectTrigger className="h-11 w-[120px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PAGE_SIZES.map((size) => (
                <SelectItem key={size} value={String(size)}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </label>
      </div>

      <div className="mb-2 flex items-center justify-between text-xs text-[var(--sea-ink-soft)]">
        <span>{t('payroll.resultLabel', { count: String(totalItems) })}</span>
        {isFetching ? <span>{t('payroll.refreshingLabel')}</span> : <span />}
      </div>

      {slips.length === 0 ? (
        <p className="m-0 py-8 text-center text-sm text-[var(--sea-ink-soft)]">
          {t('payroll.slipsEmpty')}
        </p>
      ) : (
        <Table className="min-w-[48rem]">
          <TableCaption className="sr-only">{t('payroll.tableCaption')}</TableCaption>
          <TableHeader>
            <TableRow className="border-[var(--line)] hover:bg-transparent">
              <TableHead scope="col" className="px-4 py-3">
                <button
                  type="button"
                  onClick={() => onSortChange('periode')}
                  aria-label={t('payroll.colPeriod')}
                  className="island-kicker inline-flex items-center gap-1"
                >
                  {t('payroll.colPeriod')}
                  <SortIcon active={sort === 'periode'} order={order} />
                </button>
              </TableHead>
              <TableHead scope="col" className="px-4 py-3">
                {t('payroll.colSlipNumber')}
              </TableHead>
              <TableHead scope="col" className="px-4 py-3">
                {t('payroll.colEmployee')}
              </TableHead>
              <TableHead scope="col" className="px-4 py-3">
                <button
                  type="button"
                  onClick={() => onSortChange('status')}
                  aria-label={t('payroll.colStatus')}
                  className="island-kicker inline-flex items-center gap-1"
                >
                  {t('payroll.colStatus')}
                  <SortIcon active={sort === 'status'} order={order} />
                </button>
              </TableHead>
              <TableHead scope="col" className="px-4 py-3 text-right">
                {t('payroll.colTotal')}
              </TableHead>
              <TableHead scope="col" className="px-4 py-3 text-right">
                <span className="sr-only">{t('payroll.viewDetail')}</span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody aria-busy={isFetching}>
            {slips.map((slip) => (
              <TableRow key={slip.id} className="border-[var(--line)] last:border-0">
                <TableCell className="px-4 py-3 text-[var(--sea-ink)]">
                  {slip.periodeLabel ?? slip.periode}
                </TableCell>
                <TableCell className="px-4 py-3 font-medium text-[var(--sea-ink)] break-words">
                  {slip.nomorSlip}
                </TableCell>
                <TableCell className="px-4 py-3 text-[var(--sea-ink-soft)]">
                  {slip.nama ?? '-'}
                </TableCell>
                <TableCell className="px-4 py-3 text-[var(--sea-ink-soft)]">
                  {slip.statusLabel ?? slip.status}
                </TableCell>
                <TableCell className="px-4 py-3 text-right font-medium text-[var(--sea-ink)]">
                  {slip.totalGaji}
                </TableCell>
                <TableCell className="px-4 py-3 text-right">
                  <Link
                    to="/payroll/slips/$payrollId"
                    params={{ payrollId: slip.id }}
                    className="inline-flex items-center rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-1 text-xs font-semibold text-[var(--sea-ink)] transition hover:bg-[var(--link-bg-hover)]"
                  >
                    {t('payroll.viewDetail')}
                  </Link>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <div className="mt-4 flex items-center justify-between gap-3">
        <p className="m-0 text-xs text-[var(--sea-ink-soft)]">
          {t('payroll.pageInfo', { page: String(page), total: String(safeTotalPages) })}
        </p>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => onPageChange(page - 1)}
            disabled={!canPrev}
            className="inline-flex h-9 w-9 items-center justify-center rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] text-[var(--sea-ink)] transition hover:bg-[var(--link-bg-hover)] disabled:opacity-40"
            aria-label={t('payroll.prevPage')}
          >
            <ChevronLeft className="h-4 w-4" aria-hidden="true" />
          </button>
          <button
            type="button"
            onClick={() => onPageChange(page + 1)}
            disabled={!canNext}
            className="inline-flex h-9 w-9 items-center justify-center rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] text-[var(--sea-ink)] transition hover:bg-[var(--link-bg-hover)] disabled:opacity-40"
            aria-label={t('payroll.nextPage')}
          >
            <ChevronRight className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  )
}
