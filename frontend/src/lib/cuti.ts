import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/lib/api'
import {
  cutiPageSchema,
  cutiSchema,
  type Cuti,
  type CutiPage,
  type CutiCreate,
  type CutiUpdate,
  type CutiAction,
} from '@/validations/schemas'

export const cutisQueryKey = ['cutis'] as const

/** Kolom urut yang diizinkan backend untuk daftar cuti. */
export const CUTI_SORT_FIELDS = [
  'userId',
  'jenis',
  'tanggalMulai',
  'tanggalSelesai',
  'status',
  'createdAt',
  'updatedAt',
] as const
export type CutiSortField = (typeof CUTI_SORT_FIELDS)[number]

export type CutiSortOrder = 'asc' | 'desc'

export type CutiListParams = {
  page: number
  pageSize: number
  search: string
  jenis: string | null
  status: string | null
  sort: CutiSortField
  order: CutiSortOrder
}

export const CUTI_PAGE_SIZES = [10, 25, 50, 100] as const
export type CutiPageSize = (typeof CUTI_PAGE_SIZES)[number]

/** Satu halaman daftar pengajuan cuti dari GET /api/v1/cuti. */
export async function fetchCutisPage(params: CutiListParams): Promise<CutiPage> {
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
  if (params.status) {
    query.set('status', params.status)
  }
  const payload = await apiRequest<unknown>(`/cuti?${query.toString()}`)
  return cutiPageSchema.parse(payload)
}

/** Kueri daftar pengajuan cuti berpaginasi. */
export function useCutisPage(params: CutiListParams) {
  return useQuery({
    queryKey: ['cutis', 'page', params] as const,
    queryFn: () => fetchCutisPage(params),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    retry: false,
  })
}

/** Satu pengajuan cuti dari GET /api/v1/cuti/{cutiId}. */
export async function fetchCuti(id: string): Promise<Cuti> {
  const payload = await apiRequest<unknown>(`/cuti/${id}`)
  return cutiSchema.parse(payload)
}

/** Buat pengajuan cuti baru via POST /api/v1/cuti. */
export async function createCuti(data: CutiCreate): Promise<Cuti> {
  const payload = await apiRequest<unknown>('/cuti', {
    method: 'POST',
    body: data,
  })
  return cutiSchema.parse(payload)
}

/** Ubah pengajuan cuti via PATCH /api/v1/cuti/{cutiId}. */
export async function updateCuti(id: string, data: CutiUpdate): Promise<Cuti> {
  const payload = await apiRequest<unknown>(`/cuti/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return cutiSchema.parse(payload)
}

/** Setujui pengajuan cuti via POST /api/v1/cuti/{cutiId}/approve. */
export async function approveCuti(id: string, data: CutiAction): Promise<Cuti> {
  const payload = await apiRequest<unknown>(`/cuti/${id}/approve`, {
    method: 'POST',
    body: data,
  })
  return cutiSchema.parse(payload)
}

/** Tolak pengajuan cuti via POST /api/v1/cuti/{cutiId}/reject. */
export async function rejectCuti(id: string, data: CutiAction): Promise<Cuti> {
  const payload = await apiRequest<unknown>(`/cuti/${id}/reject`, {
    method: 'POST',
    body: data,
  })
  return cutiSchema.parse(payload)
}

/** Batalkan pengajuan cuti (MENUNGGU atau DISETUJUI) via POST /api/v1/cuti/{cutiId}/cancel. */
export async function cancelCuti(id: string, data: CutiAction): Promise<Cuti> {
  const payload = await apiRequest<unknown>(`/cuti/${id}/cancel`, {
    method: 'POST',
    body: data,
  })
  return cutiSchema.parse(payload)
}
