import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { z } from 'zod'
import { apiRequest } from '@/lib/api'
import {
  honorSksPeriodePageSchema,
  honorSksRateListSchema,
  honorSksRateSchema,
  honorSksPeriodeSchema,
  type HonorSksPeriode,
  type HonorSksPeriodeCreate,
  type HonorSksPeriodeUpdate,
  type HonorSksRate,
  type HonorSksRateCreate,
  type HonorSksRateUpdate,
} from '@/validations/schemas'

export const honorSksQueryKey = ['honor-sks'] as const

/** Membaca daftar periode akademik untuk dropdown. */
export async function fetchPeriodeAkademik(): Promise<
  { id: string; tahun: number; semester: number; status: string; aktif: boolean }[]
> {
  const payload = await apiRequest<unknown>(
    '/tridharma/periode?page=1&pageSize=100&sort=tahun&order=desc',
  )
  const parsed = z
    .object({
      items: z.array(
        z
          .object({
            id: z.string(),
            tahun: z.number(),
            semester: z.number(),
            status: z.string(),
            aktif: z.boolean(),
          })
          .loose(),
      ),
    })
    .parse(payload)
  return parsed.items
}

/** Membaca seluruh tarif honor per SKS. */
export async function fetchHonorRates(): Promise<HonorSksRate[]> {
  const payload = await apiRequest<unknown>('/payroll/honor-rate')
  return honorSksRateListSchema.parse(payload)
}

export function useHonorRates() {
  return useQuery({
    queryKey: ['honor-sks', 'rates'] as const,
    queryFn: fetchHonorRates,
    staleTime: 15_000,
    retry: false,
  })
}

export async function createHonorRate(data: HonorSksRateCreate): Promise<HonorSksRate> {
  const payload = await apiRequest<unknown>('/payroll/honor-rate', {
    method: 'POST',
    body: data,
  })
  return honorSksRateSchema.parse(payload)
}

export async function updateHonorRate(
  id: string,
  data: HonorSksRateUpdate,
): Promise<HonorSksRate> {
  const payload = await apiRequest<unknown>(`/payroll/honor-rate/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return honorSksRateSchema.parse(payload)
}

/** Satu halaman beban mengajar dosen LB per periode akademik. */
export async function fetchHonorPeriode(params: {
  page: number
  pageSize: number
  sort: string
  order: string
  periodeAkademikId?: string
}): Promise<import('@/validations/schemas').HonorSksPeriodePage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort,
    order: params.order,
  })
  if (params.periodeAkademikId) {
    query.set('periodeAkademikId', params.periodeAkademikId)
  }
  const payload = await apiRequest<unknown>(`/payroll/honor-periode?${query.toString()}`)
  return honorSksPeriodePageSchema.parse(payload)
}

export function useHonorPeriode(params: {
  page: number
  pageSize: number
  sort: string
  order: string
  periodeAkademikId?: string
}) {
  return useQuery({
    queryKey: ['honor-sks', 'periode', params] as const,
    queryFn: () => fetchHonorPeriode(params),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    retry: false,
  })
}

export async function createHonorPeriode(
  data: HonorSksPeriodeCreate,
): Promise<HonorSksPeriode> {
  const payload = await apiRequest<unknown>('/payroll/honor-periode', {
    method: 'POST',
    body: data,
  })
  return honorSksPeriodeSchema.parse(payload)
}

export async function updateHonorPeriode(
  id: string,
  data: HonorSksPeriodeUpdate,
): Promise<HonorSksPeriode> {
  const payload = await apiRequest<unknown>(`/payroll/honor-periode/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return honorSksPeriodeSchema.parse(payload)
}

export async function deleteHonorPeriode(id: string): Promise<void> {
  await apiRequest<void>(`/payroll/honor-periode/${id}`, { method: 'DELETE' })
}
