import { useQuery } from '@tanstack/react-query'
import { keepPreviousData } from '@tanstack/react-query'

import { apiRequest } from '@/lib/api'

export const ARSIP_SORT_FIELDS = [
  'jenisDokumen',
  'judul',
  'berlakuSampai',
  'createdAt',
  'updatedAt',
] as const
export type ArsipSortField = (typeof ARSIP_SORT_FIELDS)[number]
export type ArsipSortOrder = 'asc' | 'desc'
export type ArsipPageSize = 10 | 25 | 50 | 100

export type ArsipRecord = {
  id: string
  userId: string
  jenisDokumen: string
  judul: string
  fileUrl: string
  fileSize: number
  mimeType: string
  keterangan: string | null
  berlakuSampai: string | null
  cloudinaryPublicId: string | null
  createdAt: string
  updatedAt: string
}

export type ArsipPage = {
  items: ArsipRecord[]
  page: number
  pageSize: number
  totalItems: number
  totalPages: number
}

export type ArsipListParams = {
  page: number
  pageSize: ArsipPageSize
  search?: string
  sort: ArsipSortField
  order: ArsipSortOrder
  userId?: string
}

export async function fetchArsip(params: ArsipListParams): Promise<ArsipPage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort,
    order: params.order,
  })
  if (params.search) query.set('search', params.search)
  if (params.userId) query.set('userId', params.userId)
  const payload = await apiRequest<unknown>(`/arsip?${query.toString()}`)
  return payload as ArsipPage
}

export function useArsip(params: ArsipListParams) {
  return useQuery({
    queryKey: ['arsip', params] as const,
    queryFn: () => fetchArsip(params),
    placeholderData: keepPreviousData,
    staleTime: 30_000,
    retry: false,
  })
}
