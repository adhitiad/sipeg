import { useCallback, useMemo, useState } from 'react'

/**
 * Kendali tabel bersama untuk daftar yang dipaginasikan server.
 *
 * Backend memiliki pagination, sorting, dan filtering, jadi komponen hanya
 * menyimpan state dan mengirim nilainya. Tiga aturan yang dijaga helper ini:
 * nilai sort harus berasal dari allowlist backend, `pageIndex` kembali ke
 * awal setiap kali sort atau filter berubah, dan perubahan `pageSize` juga
 * mengembalikan halaman ke awal.
 */

export type SortOrder = 'asc' | 'desc'

export type TableControlsOptions<SortField extends string> = {
  /** Allowlist kolom sort yang sama dengan milik backend. */
  sortFields: readonly SortField[]
  initialSort: SortField
  initialOrder?: SortOrder
  initialPageSize?: number
}

export type TableControls<SortField extends string> = {
  sort: SortField
  order: SortOrder
  page: number
  pageSize: number
  setSort: (field: SortField) => void
  setOrder: (order: SortOrder) => void
  /** Menerapkan sort beserta arahnya, lalu mengembalikan halaman ke awal. */
  applySort: (field: SortField, order: SortOrder) => void
  setPage: (page: number) => void
  setPageSize: (size: number) => void
  toggleSort: (field: SortField) => void
  /** Dipakai setiap kali filter berubah agar halaman kembali ke awal. */
  resetPage: () => void
  /** Menjaga nilai sort tetap di dalam allowlist backend. */
  isSortField: (field: string) => field is SortField
}

export function useTableControls<SortField extends string>(
  options: TableControlsOptions<SortField>,
): TableControls<SortField> {
  const { sortFields, initialSort, initialOrder, initialPageSize } = options
  const [sort, setSortState] = useState<SortField>(initialSort)
  const [order, setOrderState] = useState<SortOrder>(initialOrder ?? 'asc')
  const [page, setPageState] = useState(1)
  const [pageSize, setPageSizeState] = useState(initialPageSize ?? 20)

  const allowed = useMemo(() => new Set<string>(sortFields), [sortFields])

  const resetPage = useCallback(() => setPageState(1), [])

  const setSort = useCallback(
    (field: SortField) => {
      setSortState(field)
      setOrderState('asc')
      resetPage()
    },
    [resetPage],
  )

  const setOrder = useCallback(
    (next: SortOrder) => {
      setOrderState(next)
      resetPage()
    },
    [resetPage],
  )

  const setPage = useCallback((next: number) => setPageState(next), [])

  const applySort = useCallback(
    (field: SortField, next: SortOrder) => {
      setSortState(field)
      setOrderState(next)
      resetPage()
    },
    [resetPage],
  )

  const setPageSize = useCallback(
    (size: number) => {
      setPageSizeState(size)
      resetPage()
    },
    [resetPage],
  )

  const toggleSort = useCallback(
    (field: SortField) => {
      setSortState((current) => {
        if (current === field) {
          setOrderState((currentOrder) => (currentOrder === 'asc' ? 'desc' : 'asc'))
          return current
        }
        setOrderState('asc')
        return field
      })
      resetPage()
    },
    [resetPage],
  )

  const isSortField = useCallback(
    (field: string): field is SortField => allowed.has(field),
    [allowed],
  )

  return {
    sort,
    order,
    page,
    pageSize,
    setSort,
    setOrder,
    applySort,
    setPage,
    setPageSize,
    toggleSort,
    resetPage,
    isSortField,
  }
}

/**
 * Filter daftar memanggil `applyFilter` supaya halaman selalu kembali ke
 * awal. Nilai filter sendiri tetap milik pemanggil.
 */
export function useFilterControl() {
  const [filterVersion, setFilterVersion] = useState(0)
  const applyFilter = useCallback(() => setFilterVersion((v) => v + 1), [])
  return { filterVersion, applyFilter }
}

/** Rentang baris yang tampil untuk penomoran hasil. */
export function visibleRange(
  page: number,
  pageSize: number,
  totalItems: number,
): [number, number] {
  if (totalItems === 0) return [0, 0]
  return [(page - 1) * pageSize + 1, Math.min(page * pageSize, totalItems)]
}

/**
 * Nomor halaman yang ditampilkan. Untuk daftar panjang, halaman sekitar
 * posisi saat ini saja yang dimunculkan supaya tombol tidak menjadi baris
 * yang memenuhi lebar tabel.
 */
export function pageWindow(current: number, total: number): (number | null)[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, index) => index + 1)
  }
  const pages = new Set<number>([1, total, current, current - 1, current + 1])
  const sorted = [...pages]
    .filter((page) => page >= 1 && page <= total)
    .sort((a, b) => a - b)
  const result: (number | null)[] = []
  let previous = 0
  for (const page of sorted) {
    if (previous && page - previous > 1) {
      result.push(null)
    }
    result.push(page)
    previous = page
  }
  return result
}
