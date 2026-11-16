import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/lib/api'
import {
  kategoriStatusPegawaiListSchema,
  payrollScaleDetailSchema,
  payrollScalePageSchema,
  payrollScaleSchema,
  payrollScaleWithDetailsSchema,
  type KategoriStatusPegawai,
  type PayrollScale,
  type PayrollScaleCreate,
  type PayrollScaleDetail,
  type PayrollScaleDetailCreate,
  type PayrollScaleDetailUpdate,
  type PayrollScalePage,
  type PayrollScaleUpdate,
  type PayrollScaleWithDetails,
} from '@/validations/schemas'

export const PAYROLL_SCALE_SORT_FIELDS = [
  'kode',
  'nama',
  'berlakuMulai',
  'berlakuSelesai',
  'aktif',
  'createdAt',
  'updatedAt',
] as const
export type PayrollScaleSortField = (typeof PAYROLL_SCALE_SORT_FIELDS)[number]
export type PayrollScaleSortOrder = 'asc' | 'desc'

export type PayrollScaleListParams = {
  page: number
  pageSize: number
  search: string
  sort: PayrollScaleSortField
  order: PayrollScaleSortOrder
}

export const payrollScaleQueryKey = ['payroll-scale'] as const

/** Membaca satu halaman daftar skala gaji. */
export async function fetchPayrollScales(
  params: PayrollScaleListParams,
): Promise<PayrollScalePage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort,
    order: params.order,
  })
  if (params.search) {
    query.set('search', params.search)
  }
  const payload = await apiRequest<unknown>(`/payroll/scale?${query.toString()}`)
  return payrollScalePageSchema.parse(payload)
}

export function usePayrollScales(params: PayrollScaleListParams) {
  return useQuery({
    queryKey: ['payroll-scale', 'list', params] as const,
    queryFn: () => fetchPayrollScales(params),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    retry: false,
  })
}

/** Membaca satu skala gaji beserta baris nominalnya. */
export async function fetchPayrollScale(id: string): Promise<PayrollScaleWithDetails> {
  const payload = await apiRequest<unknown>(`/payroll/scale/${id}`)
  return payrollScaleWithDetailsSchema.parse(payload)
}

export function usePayrollScaleDetails(id: string | null) {
  return useQuery({
    queryKey: ['payroll-scale', 'detail', id] as const,
    queryFn: () => fetchPayrollScale(id!),
    enabled: id !== null,
    staleTime: 15_000,
    retry: false,
  })
}

/** Membaca taxonomy kategori status pegawai. */
export async function fetchKategoriStatus(): Promise<KategoriStatusPegawai[]> {
  const payload = await apiRequest<unknown>('/payroll/scale/kategori-status')
  return kategoriStatusPegawaiListSchema.parse(payload)
}

export function useKategoriStatus() {
  return useQuery({
    queryKey: ['payroll-scale', 'kategori-status'] as const,
    queryFn: fetchKategoriStatus,
    staleTime: 60_000,
    retry: false,
  })
}

export async function createPayrollScale(
  data: PayrollScaleCreate,
): Promise<PayrollScale> {
  const payload = await apiRequest<unknown>('/payroll/scale', {
    method: 'POST',
    body: data,
  })
  return payrollScaleSchema.parse(payload)
}

export async function updatePayrollScale(
  id: string,
  data: PayrollScaleUpdate,
): Promise<PayrollScale> {
  const payload = await apiRequest<unknown>(`/payroll/scale/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return payrollScaleSchema.parse(payload)
}

export async function createPayrollScaleDetail(
  skalaGajiId: string,
  data: PayrollScaleDetailCreate,
): Promise<PayrollScaleDetail> {
  const payload = await apiRequest<unknown>(`/payroll/scale/${skalaGajiId}/detail`, {
    method: 'POST',
    body: data,
  })
  return payrollScaleDetailSchema.parse(payload)
}

export async function updatePayrollScaleDetail(
  id: string,
  data: PayrollScaleDetailUpdate,
): Promise<PayrollScaleDetail> {
  const payload = await apiRequest<unknown>(`/payroll/scale/detail/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return payrollScaleDetailSchema.parse(payload)
}

export async function deletePayrollScaleDetail(id: string): Promise<void> {
  await apiRequest<void>(`/payroll/scale/detail/${id}`, { method: 'DELETE' })
}
