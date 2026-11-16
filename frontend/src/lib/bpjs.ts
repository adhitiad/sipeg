import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/lib/api'
import {
  bpjsKonfigurasiListSchema,
  bpjsKonfigurasiSchema,
  bpjsPesertaPageSchema,
  bpjsPesertaSchema,
  type BpjsKonfigurasi,
  type BpjsKonfigurasiCreate,
  type BpjsKonfigurasiUpdate,
  type BpjsPeserta,
  type BpjsPesertaCreate,
  type BpjsPesertaUpdate,
} from '@/validations/schemas'

export const bpjsQueryKey = ['bpjs'] as const

/** Membaca seluruh konfigurasi tarif iuran BPJS. */
export async function fetchBpjsKonfigurasi(): Promise<BpjsKonfigurasi[]> {
  const payload = await apiRequest<unknown>('/payroll/bpjs/config')
  return bpjsKonfigurasiListSchema.parse(payload)
}

export function useBpjsKonfigurasi() {
  return useQuery({
    queryKey: ['bpjs', 'config'] as const,
    queryFn: fetchBpjsKonfigurasi,
    staleTime: 15_000,
    retry: false,
  })
}

export async function createBpjsKonfigurasi(
  data: BpjsKonfigurasiCreate,
): Promise<BpjsKonfigurasi> {
  const payload = await apiRequest<unknown>('/payroll/bpjs/config', {
    method: 'POST',
    body: data,
  })
  return bpjsKonfigurasiSchema.parse(payload)
}

export async function updateBpjsKonfigurasi(
  id: string,
  data: BpjsKonfigurasiUpdate,
): Promise<BpjsKonfigurasi> {
  const payload = await apiRequest<unknown>(`/payroll/bpjs/config/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return bpjsKonfigurasiSchema.parse(payload)
}

/** Satu halaman kepesertaan BPJS. */
export async function fetchBpjsPeserta(params: {
  page: number
  pageSize: number
  sort: string
  order: string
}): Promise<import('@/validations/schemas').BpjsPesertaPage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort,
    order: params.order,
  })
  const payload = await apiRequest<unknown>(`/payroll/bpjs/peserta?${query.toString()}`)
  return bpjsPesertaPageSchema.parse(payload)
}

export function useBpjsPeserta(params: {
  page: number
  pageSize: number
  sort: string
  order: string
}) {
  return useQuery({
    queryKey: ['bpjs', 'peserta', params] as const,
    queryFn: () => fetchBpjsPeserta(params),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    retry: false,
  })
}

export async function createBpjsPeserta(data: BpjsPesertaCreate): Promise<BpjsPeserta> {
  const payload = await apiRequest<unknown>('/payroll/bpjs/peserta', {
    method: 'POST',
    body: data,
  })
  return bpjsPesertaSchema.parse(payload)
}

export async function updateBpjsPeserta(
  id: string,
  data: BpjsPesertaUpdate,
): Promise<BpjsPeserta> {
  const payload = await apiRequest<unknown>(`/payroll/bpjs/peserta/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return bpjsPesertaSchema.parse(payload)
}
