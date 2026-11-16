import { useMemo } from 'react'
import { DataTable } from '@/components/DataTable'
import type { ColumnDef } from '@tanstack/react-table'
import type { ArsipRecord, ArsipPageSize } from '@/lib/arsip'
import { ARSIP_SORT_FIELDS } from '@/lib/arsip'

type ArsipTableProps = {
  items: ArsipRecord[]
  page: number
  pageSize: ArsipPageSize
  totalPages: number
  totalItems: number
  isFetching: boolean
  sort: string
  order: 'asc' | 'desc'
  onPageChange: (page: number) => void
  onPageSizeChange: (size: ArsipPageSize) => void
  onSortChange: (field: string, order: 'asc' | 'desc') => void
  labels: Record<string, string>
}

export default function ArsipTable({
  items,
  page,
  pageSize,
  totalPages,
  totalItems,
  isFetching,
  sort,
  order,
  onPageChange,
  onPageSizeChange,
  onSortChange,
  labels,
}: ArsipTableProps) {
  const columns = useMemo<ColumnDef<ArsipRecord, unknown>[]>(
    () => [
      { id: 'judul', accessorFn: (r) => r.judul, header: labels.judul },
      {
        id: 'jenisDokumen',
        accessorFn: (r) => r.jenisDokumen,
        header: labels.jenisDokumen,
      },
      { id: 'fileSize', accessorFn: (r) => r.fileSize, header: labels.fileSize },
      {
        id: 'berlakuSampai',
        accessorFn: (r) => r.berlakuSampai,
        header: labels.berlakuSampai,
      },
      { id: 'createdAt', accessorFn: (r) => r.createdAt, header: labels.createdAt },
    ],
    [labels],
  )

  return (
    <DataTable
      columns={columns}
      data={items}
      getRowId={(row) => row.id}
      sortFields={ARSIP_SORT_FIELDS as unknown as string[]}
      sort={sort as (typeof ARSIP_SORT_FIELDS)[number]}
      order={order}
      onSortChange={(f, o) => onSortChange(f, o)}
      page={page}
      pageSize={pageSize}
      totalItems={totalItems}
      totalPages={totalPages}
      onPageChange={onPageChange}
      onPageSizeChange={(size) => onPageSizeChange(size as ArsipPageSize)}
      pageSizeOptions={[10, 25, 50, 100]}
      isFetching={isFetching}
      labels={labels as never}
    />
  )
}

export type { ArsipPageSize }
