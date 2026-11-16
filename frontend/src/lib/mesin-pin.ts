import { useQuery } from '@tanstack/react-query'
import { API_BASE_URL, ApiError, apiRequest } from '@/lib/api'
import { getAuthToken } from '@/lib/auth-token'
import {
  alpaSuspectsResponseSchema,
  derivePresensiResponseSchema,
  mesinAbsensiPinCreateSchema,
  mesinAbsensiPinListSchema,
  mesinAbsensiPinSchema,
  pinImportResultSchema,
  presensiUpsertResponseSchema,
  type AlpaSuspectsResponse,
  type DerivePresensiResponse,
  type MesinAbsensiPin,
  type MesinAbsensiPinCreate,
  type PinImportResult,
  type PresensiUpsertRequest,
  type PresensiUpsertResponse,
} from '@/validations/schemas'

export const mesinPinQueryKey = ['mesin-pin'] as const

/** Membaca seluruh pemetaan PIN mesin (cakupan sesuai permission). */
export async function fetchMesinPins(): Promise<MesinAbsensiPin[]> {
  const payload = await apiRequest<unknown>('/attendance/mesin-pin')
  return mesinAbsensiPinListSchema.parse(payload).items
}

export function useMesinPins() {
  return useQuery({
    queryKey: mesinPinQueryKey,
    queryFn: fetchMesinPins,
    staleTime: 15_000,
    retry: false,
  })
}

export async function createMesinPin(
  data: MesinAbsensiPinCreate,
): Promise<MesinAbsensiPin> {
  const parsed = mesinAbsensiPinCreateSchema.parse(data)
  const payload = await apiRequest<unknown>('/attendance/mesin-pin', {
    method: 'POST',
    body: parsed,
  })
  return mesinAbsensiPinSchema.parse(payload)
}

export async function deleteMesinPin(id: string): Promise<void> {
  await apiRequest<void>(`/attendance/mesin-pin/${id}`, { method: 'DELETE' })
}

export const MAX_PIN_IMPORT_BYTES = 2 * 1024 * 1024

/** Mengunggah berkas CSV/XLSX pemetaan PIN via POST /attendance/mesin-pin/import. */
export async function importMesinPins(file: File): Promise<PinImportResult> {
  const form = new FormData()
  form.append('file', file)
  const headers: Record<string, string> = { Accept: 'application/json' }
  const token = getAuthToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  const response = await fetch(`${API_BASE_URL}/attendance/mesin-pin/import`, {
    method: 'POST',
    headers,
    body: form,
  })
  const body: unknown = await response.json().catch(() => undefined)
  if (!response.ok) {
    throw new ApiError({
      status: response.status,
      code: 'import_failed',
      message: `Impor gagal (${response.status})`,
    })
  }
  return pinImportResultSchema.parse(body)
}

/** Mengunduh templat CSV impor pemetaan PIN. */
export async function downloadMesinPinTemplate(): Promise<void> {
  const headers: Record<string, string> = {}
  const token = getAuthToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  const response = await fetch(`${API_BASE_URL}/attendance/mesin-pin/import-template`, {
    headers,
  })
  if (!response.ok) {
    throw new Error(`Templat gagal diunduh (${response.status})`)
  }
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = 'mesin-pin-import-template.csv'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

/** Menurunkan status harian Presensi dari AttendanceEvent pada rentang tanggal. */
export async function derivePresensi(
  from: string,
  to: string,
): Promise<DerivePresensiResponse> {
  const payload = await apiRequest<unknown>('/attendance/presensi/derive', {
    method: 'POST',
    body: { from, to },
  })
  return derivePresensiResponseSchema.parse(payload)
}

/** Mengisi atau mengubah satu baris Presensi secara manual. */
export async function upsertPresensi(
  data: PresensiUpsertRequest,
): Promise<PresensiUpsertResponse> {
  const payload = await apiRequest<unknown>('/attendance/presensi', {
    method: 'PUT',
    body: data,
  })
  return presensiUpsertResponseSchema.parse(payload)
}

/** Membaca laporan pegawai dengan alpa pada bulan berturut-turut. */
export async function fetchAlpaSuspects(
  until: string,
  months: number,
): Promise<AlpaSuspectsResponse> {
  const query = new URLSearchParams({ until, months: String(months) })
  const payload = await apiRequest<unknown>(
    `/attendance/alpa-suspects?${query.toString()}`,
  )
  return alpaSuspectsResponseSchema.parse(payload)
}
