import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/lib/api'
import {
  payrollPageSchema,
  payrollScheduleListSchema,
  payrollScheduleSchema,
  payrollSlipSchema,
  type PayrollPage,
  type PayrollSchedule,
  type PayrollSlip,
} from '@/validations/schemas'

export const PAYROLL_SORT_FIELDS = [
  'periode',
  'status',
  'createdAt',
  'updatedAt',
] as const
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
