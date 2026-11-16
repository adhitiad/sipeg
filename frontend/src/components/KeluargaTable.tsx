import { useMemo } from 'react'
import { DataTable } from '@/components/DataTable'
import type { ColumnDef } from '@tanstack/react-table'
import type { KeluargaRecord, KeluargaPageSize } from '@/lib/keluarga'
export type { KeluargaPageSize }
import { KELUARGA_SORT_FIELDS } from '@/lib/keluarga'

type KeluargaTableProps = {
  items: KeluargaRecord[]
  page: number
  pageSize: KeluargaPageSize
  totalPages: number
  totalItems: number
  isFetching: boolean
  sort: string
  order: 'asc' | 'desc'
  onPageChange: (page: number) => void
  onPageSizeChange: (size: KeluargaPageSize) => void
  onSortChange: (field: string, order: 'asc' | 'desc') => void
  labels: Record<string, string>
}

export default function KeluargaTable({
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
}: KeluargaTableProps) {
  const columns = useMemo<ColumnDef<KeluargaRecord, unknown>[]>(
    () => [
      { id: 'nama', accessorFn: (r) => r.nama, header: labels.nama },
      { id: 'hubungan', accessorFn: (r) => r.hubungan, header: labels.hubungan },
      {
        id: 'tanggalLahir',
        accessorFn: (r) => r.tanggalLahir,
        header: labels.tanggalLahir,
      },
      {
        id: 'statusTanggungan',
        accessorFn: (r) => r.statusTanggungan,
        header: labels.statusTanggungan,
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
      sortFields={KELUARGA_SORT_FIELDS as unknown as string[]}
      sort={sort as (typeof KELUARGA_SORT_FIELDS)[number]}
      order={order}
      onSortChange={(f, o) => onSortChange(f, o)}
      page={page}
      pageSize={pageSize}
      totalItems={totalItems}
      totalPages={totalPages}
      onPageChange={onPageChange}
      onPageSizeChange={(size) => onPageSizeChange(size as KeluargaPageSize)}
      pageSizeOptions={[10, 25, 50, 100]}
      isFetching={isFetching}
      labels={labels as never}
    />
  )
}
