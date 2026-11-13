import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { z } from 'zod'
import { apiRequest } from '@/lib/api'
import {
  employeePageSchema,
  employeeSchema,
  type Employee,
  type EmployeePage,
  type EmployeeUpdate,
} from '@/validations/schemas'

/** Kolom urut yang diizinkan backend untuk daftar pegawai. Kolom terenkripsi (nip) tidak dapat diurutkan. */
export const EMPLOYEE_SORT_FIELDS = ['nama', 'createdAt', 'updatedAt'] as const
export type EmployeeSortField = (typeof EMPLOYEE_SORT_FIELDS)[number]

export type EmployeeSortOrder = 'asc' | 'desc'

export type EmployeeListParams = {
  page: number
  pageSize: number
  search: string
  sort: EmployeeSortField
  order: EmployeeSortOrder
  homebaseId: string | null
}

export const employeesQueryKey = (params: EmployeeListParams) =>
  ['employees', params] as const

/**
 * Membaca daftar pegawai dalam cakupan pemohon. Cakupan unit ditegakkan
 * server; `homebaseId` di sini hanya filter pilihan, bukan kontrol akses.
 */
export async function fetchEmployees(params: EmployeeListParams): Promise<EmployeePage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort,
    order: params.order,
  })
  if (params.search) {
    query.set('search', params.search)
  }
  if (params.homebaseId) {
    query.set('homebaseId', params.homebaseId)
  }

  const payload = await apiRequest<unknown>(`/employees?${query.toString()}`)
  return employeePageSchema.parse(payload)
}

/** Satu profil pegawai dari GET /api/v1/employees/{employeeId}. */
export async function fetchEmployee(id: string): Promise<Employee> {
  const payload = await apiRequest<unknown>(`/employees/${id}`)
  return employeeSchema.parse(payload)
}

/** Perbarui field profil yang diizinkan via PATCH /api/v1/employees/{employeeId}. */
export async function updateEmployee(
  id: string,
  data: EmployeeUpdate,
): Promise<Employee> {
  const payload = await apiRequest<unknown>(`/employees/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return employeeSchema.parse(payload)
}

/** Tetapkan peran pegawai via PUT /api/v1/employees/{employeeId}/role (khusus admin). */
export async function assignEmployeeRole(
  id: string,
  roleName: 'admin' | 'operator' | 'atasan' | 'pegawai',
): Promise<{ employeeId: string; roleName: string }> {
  const payload = await apiRequest<unknown>(`/employees/${id}/role`, {
    method: 'PUT',
    body: { roleName },
  })
  return z.object({ employeeId: z.string(), roleName: z.string() }).parse(payload)
}

/** Kueri satu profil pegawai. */
export function useEmployee(id: string) {
  return useQuery({
    queryKey: ['employee', id] as const,
    queryFn: () => fetchEmployee(id),
    enabled: !!id,
    staleTime: 15_000,
    retry: false,
  })
}

/** Kueri daftar pegawai. `keepPreviousData` dipakai agar pergantian halaman
 * dan filter tidak mengosongkan tabel sebelum baris baru tiba. */
export function useEmployees(params: EmployeeListParams) {
  return useQuery({
    queryKey: employeesQueryKey(params),
    queryFn: () => fetchEmployees(params),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    retry: false,
  })
}
