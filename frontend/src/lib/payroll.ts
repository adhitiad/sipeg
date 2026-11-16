import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { API_BASE_URL, apiRequest } from '@/lib/api'
import { getAuthToken } from '@/lib/auth-token'
import {
  payrollPageSchema,
  payrollScheduleListSchema,
  payrollScheduleSchema,
  payrollSlipSchema,
  type PayrollPage,
  type PayrollSchedule,
  type PayrollSlip,
} from '@/validations/schemas'

/**
 * Kolom urut yang diizinkan backend untuk daftar slip gaji. Backend hanya
 * menerima createdAt, periode, dan tanggalGaji; kolom lain ditolak dengan
 * 400 invalid_query.
 */
export const PAYROLL_SORT_FIELDS = ['createdAt', 'periode', 'tanggalGaji'] as const
export type PayrollSortField = (typeof PAYROLL_SORT_FIELDS)[number]
export type PayrollSortOrder = 'asc' | 'desc'

export type PayrollListParams = {
  page: number
  pageSize: number
  periode: string
  status: string
  sort: PayrollSortField
  order: PayrollSortOrder
}

export const payrollSlipsQueryKey = (params: PayrollListParams) =>
  ['payroll', 'slips', params] as const

export const payrollSlipQueryKey = (id: string) => ['payroll', 'slip', id] as const

/** Mengunduh slip gaji sebagai PDF (token hanya di memori, jadi lewat fetch + blob). */
export async function downloadSlipPdf(id: string, nomorSlip: string): Promise<void> {
  const headers: Record<string, string> = {}
  const token = getAuthToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  const response = await fetch(`${API_BASE_URL}/payroll/slips/${id}/pdf`, { headers })
  if (!response.ok) {
    throw new Error(`PDF slip gagal diunduh (${response.status})`)
  }
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `slip-${nomorSlip}.pdf`
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

/** Membuat atau mengganti slip DRAF lewat mesin hitung (payroll:manage). */
export async function generateSlip(data: {
  userId: string
  periode: string
  kategoriStatusId?: string
  tingkat?: number
}): Promise<PayrollSlip> {
  const payload = await apiRequest<unknown>('/payroll/slips/generate', {
    method: 'POST',
    body: data,
  })
  return payrollSlipSchema.parse(payload)
}

export async function fetchPayrollSlips(params: PayrollListParams): Promise<PayrollPage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort,
    order: params.order,
  })
  if (params.periode) query.set('periode', params.periode)
  if (params.status) query.set('status', params.status)
  const payload = await apiRequest<unknown>(`/payroll/slips?${query.toString()}`)
  return payrollPageSchema.parse(payload)
}

export async function fetchPayrollSlip(id: string): Promise<PayrollSlip> {
  const payload = await apiRequest<unknown>(`/payroll/slips/${id}`)
  return payrollSlipSchema.parse(payload)
}

export async function confirmPayrollPayment(
  id: string,
  data: {
    referensiTransaksi?: string | null
    pelakuPengesahan: string
    tanggalBayar: string
    keterangan?: string | null
  },
): Promise<PayrollSlip> {
  const payload = await apiRequest<unknown>(`/payroll/slips/${id}/payment`, {
    method: 'POST',
    body: data,
  })
  return payrollSlipSchema.parse(payload)
}

/** Menerbitkan slip DRAF menjadi TERBIT (prasyarat konfirmasi pembayaran). */
export async function publishSlip(id: string): Promise<PayrollSlip> {
  const payload = await apiRequest<unknown>(`/payroll/slips/${id}/terbit`, {
    method: 'POST',
    body: {},
  })
  return payrollSlipSchema.parse(payload)
}

export function usePayrollSlips(params: PayrollListParams) {
  return useQuery({
    queryKey: payrollSlipsQueryKey(params),
    queryFn: () => fetchPayrollSlips(params),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    retry: false,
  })
}

export function usePayrollSlip(id: string) {
  return useQuery({
    queryKey: payrollSlipQueryKey(id),
    queryFn: () => fetchPayrollSlip(id),
    enabled: !!id,
    staleTime: 15_000,
    retry: false,
  })
}

export const payrollSchedulesQueryKey = ['payroll', 'schedules'] as const

export async function fetchPayrollSchedules(): Promise<PayrollSchedule[]> {
  const payload = await apiRequest<unknown>('/payroll/schedules')
  return payrollScheduleListSchema.parse(payload).items
}

export async function upsertPayrollSchedule(
  unitId: string,
  tanggalGajian: number,
): Promise<PayrollSchedule> {
  const payload = await apiRequest<unknown>(`/payroll/schedules/${unitId}`, {
    method: 'PUT',
    body: { tanggalGajian },
  })
  return payrollScheduleSchema.parse(payload)
}

export function usePayrollSchedules() {
  return useQuery({
    queryKey: payrollSchedulesQueryKey,
    queryFn: fetchPayrollSchedules,
    staleTime: 30_000,
    retry: false,
  })
}
