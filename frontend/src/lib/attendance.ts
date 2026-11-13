import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/lib/api'
import {
  attendancePageSchema,
  attendanceSummarySchema,
  unitDayRecordListSchema,
  type AttendancePage,
  type AttendanceSummary,
  type UnitDayRecordList,
} from '@/validations/schemas'

/** Ukuran halaman default untuk daftar kehadiran. */
export const ATTENDANCE_PAGE_SIZE = 20

/** Jumlah baris per halaman pada daftar kehadiran tim. */
export const TEAM_ATTENDANCE_PAGE_SIZE = 20

export type AttendanceListParams = {
  page: number
  pageSize: number
  from: string
  to: string
  search: string
  sort: 'date'
  order: 'asc' | 'desc'
  status: string | null
}

/** Query key untuk daftar kehadiran. */
export const attendanceQueryKey = (params: AttendanceListParams) =>
  ['attendance', params] as const

/** Query key untuk ringkasan kehadiran unit. */
export const attendanceSummaryQueryKey = (range: { from: string; to: string }) =>
  ['attendance', 'unit-summary', range] as const

/** Query key untuk kehadiran tim per tanggal. */
export const attendanceUnitDayQueryKey = (date: string) =>
  ['attendance', 'unit-records', date] as const

export async function fetchAttendanceSummary(range: {
  from: string
  to: string
}): Promise<AttendanceSummary> {
  const payload = await apiRequest<unknown>(
    `/attendance/unit-summary?from=${range.from}&to=${range.to}`,
  )
  return attendanceSummarySchema.parse(payload)
}

/**
 * Kueri ringkasan kehadiran unit dalam rentang tanggal tertentu. Backend
 * mengembalikan keenam kode status per hari yang memiliki catatan. Setel
 * `enabled` ke `false` bila rentang tanggal belum valid.
 */
export function useAttendanceSummary(
  range: { from: string; to: string },
  enabled = true,
) {
  return useQuery({
    queryKey: attendanceSummaryQueryKey(range),
    queryFn: () => fetchAttendanceSummary(range),
    staleTime: 15_000,
    retry: false,
    enabled,
  })
}

export async function fetchUnitDay(date: string): Promise<UnitDayRecordList> {
  const payload = await apiRequest<unknown>(`/attendance/unit-records?date=${date}`)
  return unitDayRecordListSchema.parse(payload)
}

/**
 * Kueri kehadiran tim per tanggal. Backend mengembalikan seluruh pegawai
 * aktif dalam cakupan (LEFT JOIN ke Presensi); status null berarti belum
 * tercatat. Setel `enabled` ke `false` bila tanggal belum valid.
 */
export function useUnitDay(date: string, enabled = true) {
  return useQuery({
    queryKey: attendanceUnitDayQueryKey(date),
    queryFn: () => fetchUnitDay(date),
    staleTime: 15_000,
    retry: false,
    enabled,
  })
}

/**
 * Membaca daftar catatan kehadiran dalam rentang tanggal tertentu.
 * Parameter `from` dan `to` format `YYYY-MM-DD`. Rentang maksimal 365 hari.
 */
export async function fetchAttendanceRecords(
  params: AttendanceListParams,
): Promise<AttendancePage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    from: params.from,
    to: params.to,
    sort: params.sort,
    order: params.order,
  })
  if (params.search) {
    query.set('search', params.search)
  }
  if (params.status) {
    query.set('status', params.status)
  }

  const payload = await apiRequest<unknown>(`/attendance/my-records?${query.toString()}`)
  return attendancePageSchema.parse(payload)
}

/**
 * Kueri daftar kehadiran. `keepPreviousData` dipakai agar pergantian halaman
 * dan filter tidak mengosongkan tabel sebelum baris baru tiba. Setel
 * `enabled` ke `false` bila rentang tanggal belum valid.
 */
export function useAttendanceRecords(params: AttendanceListParams, enabled = true) {
  return useQuery({
    queryKey: attendanceQueryKey(params),
    queryFn: () => fetchAttendanceRecords(params),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    retry: false,
    enabled,
  })
}
