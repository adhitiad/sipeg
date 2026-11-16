import { useMemo } from 'react'
import { Link } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import { DataTable } from '@/components/DataTable'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useMessages } from '@/i18n/locale-context'
import { PAYROLL_SORT_FIELDS } from '@/lib/payroll'
import type { PayrollSortField, PayrollSortOrder } from '@/lib/payroll'
import { visibleRange } from '@/lib/table-state'
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
  sort: PayrollSortField
  order: PayrollSortOrder
  onPageChange: (page: number) => void
  onPageSizeChange: (size: PayrollPageSize) => void
  onSearchChange: (value: string) => void
  onStatusChange: (value: string | null) => void
  onSortChange: (field: PayrollSortField, order: PayrollSortOrder) => void
}

/**
 * Tabel slip gaji. Backend memegang penyaringan, pengurutan, dan pemotongan
 * baris, jadi tabel ini hanya meneruskan pilihan pengguna ke mesin tabel
 * bersama. Kolom status tidak dapat diurutkan karena di luar allowlist
 * backend.
 */
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
  const [from, to] = visibleRange(page, pageSize, totalItems)

  const columns = useMemo<ColumnDef<PayrollSlip, unknown>[]>(
    () => [
      {
        id: 'periode',
        accessorFn: (row) => row.periode,
        header: t('payroll.colPeriod'),
        cell: ({ row }) => {
          const isThr = row.original.pendapatan?.some((item) => item.jenis === 'THR')
          return (
            <span className="inline-flex items-center gap-2 whitespace-normal text-[var(--sea-ink)]">
              {row.original.periodeLabel ?? row.original.periode}
              {isThr ? (
                <span className="inline-flex items-center rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-[var(--sea-ink)]">
                  {t('payroll.thrBadge')}
                </span>
              ) : null}
            </span>
          )
        },
      },
      {
        id: 'nomorSlip',
        accessorFn: (row) => row.nomorSlip,
        header: t('payroll.colSlipNumber'),
        cell: ({ row }) => (
          <span className="font-medium break-words whitespace-normal">
            {row.original.nomorSlip}
          </span>
        ),
      },
      {
        id: 'nama',
        accessorFn: (row) => row.nama ?? '',
        header: t('payroll.colEmployee'),
        cell: ({ row }) => (
          <span className="whitespace-normal text-[var(--sea-ink-soft)]">
            {row.original.nama ?? '-'}
          </span>
        ),
      },
      {
        id: 'status',
        accessorFn: (row) => row.status,
        header: t('payroll.colStatus'),
        cell: ({ row }) => (
          <span className="whitespace-normal text-[var(--sea-ink-soft)]">
            {row.original.statusLabel ?? row.original.status}
          </span>
        ),
      },
      {
        id: 'totalGaji',
        accessorFn: (row) => row.totalGaji,
        header: t('payroll.colTotal'),
        cell: ({ row }) => (
          <span className="font-medium whitespace-normal text-[var(--sea-ink)]">
            {row.original.totalGaji}
          </span>
        ),
      },
      {
        id: 'aksi',
        header: () => <span className="sr-only">{t('payroll.viewDetail')}</span>,
        cell: ({ row }) => (
          <Link
            to="/payroll/slips/$payrollId"
            params={{ payrollId: row.original.id }}
            className="inline-flex items-center rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-1 text-xs font-semibold text-[var(--sea-ink)] transition hover:bg-[var(--link-bg-hover)]"
          >
            {t('payroll.viewDetail')}
          </Link>
        ),
      },
    ],
    [t],
  )

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
        <label className="grid gap-1">
          <span className="text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase">
            {t('payroll.searchLabel')}
          </span>
          <Input
            type="search"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder={t('payroll.searchPlaceholder')}
            className="h-11"
          />
        </label>
        <label className="grid gap-1">
          <span className="text-xs font-semibold tracking-[0.16em] text-[var(--kicker)] uppercase">
            {t('payroll.filterStatus')}
          </span>
          <Select
            value={status ?? 'ALL'}
            onValueChange={(value) => onStatusChange(value === 'ALL' ? null : value)}
          >
            <SelectTrigger
              className="h-11 w-[180px]"
              aria-label={t('payroll.filterStatus')}
            >
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
      </div>

      <DataTable
        columns={columns}
        data={slips}
        getRowId={(row) => row.id}
        sortFields={PAYROLL_SORT_FIELDS}
        sort={sort}
        order={order}
        onSortChange={onSortChange}
        page={page}
        pageSize={pageSize}
        totalItems={totalItems}
        totalPages={totalPages}
        onPageChange={onPageChange}
        onPageSizeChange={(size) => onPageSizeChange(size as PayrollPageSize)}
        pageSizeOptions={PAGE_SIZES}
        isFetching={isFetching}
        labels={{
          caption: t('payroll.tableCaption'),
          loadingLabel: t('payroll.refreshingLabel'),
          emptyTitle: t('payroll.slipsEmpty'),
          emptyDescription: t('common.emptyDescription'),
          resultLabel: t('common.resultCount', {
            from: String(from),
            to: String(to),
            total: String(totalItems),
          }),
          pageInfo: t('common.pageInfo'),
          prevPage: t('common.prevPage'),
          nextPage: t('common.nextPage'),
          pageSizeLabel: t('common.pageSize'),
          itemsPerPage: t('common.itemsPerPage'),
          sortBy: t('common.sortBy'),
        }}
      />
    </div>
  )
}
