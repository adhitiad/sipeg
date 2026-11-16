import { API_BASE_URL, ApiError } from '@/lib/api'
import { getAuthToken } from '@/lib/auth-token'
import {
  apiErrorEnvelopeSchema,
  employeeImportResultSchema,
  type EmployeeImportResult,
} from '@/validations/schemas'

const MAX_FILE_BYTES = 5 * 1024 * 1024

/** Batas ukuran berkas impor yang diterima backend. */
export function employeeImportFileTooLarge(file: File): boolean {
  return file.size > MAX_FILE_BYTES
}

/** Mengunggah berkas pegawai via POST /api/v1/employees/import. */
export async function importEmployees(file: File): Promise<EmployeeImportResult> {
  const form = new FormData()
  form.append('file', file)
  const headers: Record<string, string> = { Accept: 'application/json' }
  const token = getAuthToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  const response = await fetch(`${API_BASE_URL}/employees/import`, {
    method: 'POST',
    headers,
    body: form,
  })
  const body: unknown = await response.json().catch(() => undefined)
  if (!response.ok) {
    const parsed = apiErrorEnvelopeSchema.safeParse(body)
    const code = parsed.success ? parsed.data.error.code : 'internal_error'
    const message = parsed.success ? (parsed.data.error.message ?? code) : code
    const rows = parsed.success ? parsed.data.error.rows : undefined
    throw new ApiError({ status: response.status, code, message, rows })
  }
  return employeeImportResultSchema.parse(body)
}

/** Mengunduh template CSV impor pegawai dari GET /api/v1/employees/import-template.csv. */
export async function downloadEmployeeImportTemplate(): Promise<void> {
  const headers: Record<string, string> = {}
  const token = getAuthToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  const response = await fetch(`${API_BASE_URL}/employees/import-template.csv`, {
    headers,
  })
  if (!response.ok) {
    throw new ApiError({
      status: response.status,
      code: 'internal_error',
      message: `Template gagal diunduh (${response.status})`,
    })
  }
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = 'employees-import-template.csv'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

/** Mengunduh ekspor CSV pegawai dari GET /api/v1/employees/export. */
export async function downloadEmployeeExport(): Promise<void> {
  const headers: Record<string, string> = {}
  const token = getAuthToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  const response = await fetch(`${API_BASE_URL}/employees/export`, {
    headers,
  })
  if (!response.ok) {
    throw new ApiError({
      status: response.status,
      code: 'internal_error',
      message: `Ekspor gagal diunduh (${response.status})`,
    })
  }
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = 'employees-export.csv'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}
