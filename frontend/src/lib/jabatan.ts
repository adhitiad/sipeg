import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/lib/api'
import {
  jabatanPageSchema,
  jabatanSchema,
  type Jabatan,
  type JabatanCreate,
  type JabatanPage,
  type JabatanUpdate,
} from '@/validations/schemas'

export const jabatansQueryKey = ['jabatans'] as const

export const JABATAN_KINDS = ['STRUKTURAL', 'FUNGSIONAL'] as const

/** Kolom urut yang diizinkan backend untuk daftar jabatan. */
export const JABATAN_SORT_FIELDS = ['nama', 'jenis', 'createdAt', 'updatedAt'] as const
export type JabatanSortField = (typeof JABATAN_SORT_FIELDS)[number]

export type JabatanSortOrder = 'asc' | 'desc'

export type JabatanListParams = {
  page: number
  pageSize: number
  search: string
  jenis: string | null
  sort: JabatanSortField
  order: JabatanSortOrder
}

/** Membaca seluruh master jabatan aktif. Hasil dibatasi 100 baris per halaman. */
export async function fetchJabatans(): Promise<Jabatan[]> {
  const payload = await apiRequest<unknown>(
    '/jabatan?page=1&pageSize=100&sort=nama&order=asc',
  )
  return jabatanPageSchema.parse(payload).items
}

/** Satu halaman daftar jabatan dari GET /api/v1/jabatan. */
export async function fetchJabatansPage(params: JabatanListParams): Promise<JabatanPage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort,
    order: params.order,
  })
  if (params.search) {
    query.set('search', params.search)
  }
  if (params.jenis) {
    query.set('jenis', params.jenis)
  }
  const payload = await apiRequest<unknown>(`/jabatan?${query.toString()}`)
  return jabatanPageSchema.parse(payload)
}

/** Kueri daftar jabatan berpaginasi. */
export function useJabatansPage(params: JabatanListParams) {
  return useQuery({
    queryKey: ['jabatans', 'page', params] as const,
    queryFn: () => fetchJabatansPage(params),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    retry: false,
  })
}

/** Satu jabatan dari GET /api/v1/jabatan/{jabatanId}. */
export async function fetchJabatan(id: string): Promise<Jabatan> {
  const payload = await apiRequest<unknown>(`/jabatan/${id}`)
  return jabatanSchema.parse(payload)
}

export async function createJabatan(data: JabatanCreate): Promise<Jabatan> {
  const payload = await apiRequest<unknown>('/jabatan', { method: 'POST', body: data })
  return jabatanSchema.parse(payload)
}

export async function updateJabatan(id: string, data: JabatanUpdate): Promise<Jabatan> {
  const payload = await apiRequest<unknown>(`/jabatan/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return jabatanSchema.parse(payload)
}

export async function archiveJabatan(id: string): Promise<void> {
  await apiRequest<void>(`/jabatan/${id}`, { method: 'DELETE' })
}
