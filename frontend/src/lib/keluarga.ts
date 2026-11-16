import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { keepPreviousData } from '@tanstack/react-query'

import { apiRequest } from '@/lib/api'

export const KELUARGA_SORT_FIELDS = [
  'nama',
  'tanggalLahir',
  'statusTanggungan',
  'createdAt',
  'updatedAt',
] as const
export type KeluargaSortField = (typeof KELUARGA_SORT_FIELDS)[number]
export type KeluargaSortOrder = 'asc' | 'desc'

export type KeluargaPageSize = 10 | 25 | 50 | 100

export type KeluargaRecord = {
  id: string
  userId: string
  hubungan: 'SUAMI_ISTRI' | 'ANAK'
  nama: string
  nik: string
  tanggalLahir: string
  pekerjaan: string | null
  statusTanggungan: 'TANGGUNGAN' | 'BUKAN_TANGGUNGAN'
  createdAt: string
  updatedAt: string
}

export type KeluargaPage = {
  items: KeluargaRecord[]
  page: number
  pageSize: number
  totalItems: number
  totalPages: number
}

export type KeluargaListParams = {
  page: number
  pageSize: KeluargaPageSize
  search?: string
  sort: KeluargaSortField
  order: KeluargaSortOrder
  userId?: string
}

export type KeluargaCreate = {
  userId: string
  hubungan: 'SUAMI_ISTRI' | 'ANAK'
  nama: string
  nik: string
  tanggalLahir: string
  pekerjaan?: string | null
  statusTanggungan: 'TANGGUNGAN' | 'BUKAN_TANGGUNGAN'
}

export type KeluargaUpdate = {
  nama?: string
  nik?: string
  tanggalLahir?: string
  pekerjaan?: string | null
  statusTanggungan?: 'TANGGUNGAN' | 'BUKAN_TANGGUNGAN'
}

export async function fetchKeluarga(params: KeluargaListParams): Promise<KeluargaPage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort,
    order: params.order,
  })
  if (params.search) query.set('search', params.search)
  if (params.userId) query.set('userId', params.userId)
  const payload = await apiRequest<unknown>(`/keluarga?${query.toString()}`)
  return payload as KeluargaPage
}

export function useKeluarga(params: KeluargaListParams) {
  return useQuery({
    queryKey: ['keluarga', params] as const,
    queryFn: () => fetchKeluarga(params),
    placeholderData: keepPreviousData,
    staleTime: 30_000,
    retry: false,
  })
}

export async function createKeluarga(payload: KeluargaCreate): Promise<KeluargaRecord> {
  return apiRequest<KeluargaRecord>('/keluarga', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function updateKeluarga(
  id: string,
  payload: KeluargaUpdate,
): Promise<KeluargaRecord> {
  return apiRequest<KeluargaRecord>(`/keluarga/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

export async function deleteKeluarga(id: string): Promise<void> {
  await apiRequest<void>(`/keluarga/${id}`, { method: 'DELETE' })
}

export function useCreateKeluarga() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: createKeluarga,
    onSuccess: () => client.invalidateQueries({ queryKey: ['keluarga'] }),
  })
}

export function useUpdateKeluarga() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: KeluargaUpdate }) =>
      updateKeluarga(id, payload),
    onSuccess: () => client.invalidateQueries({ queryKey: ['keluarga'] }),
  })
}

export function useDeleteKeluarga() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: deleteKeluarga,
    onSuccess: () => client.invalidateQueries({ queryKey: ['keluarga'] }),
  })
}
