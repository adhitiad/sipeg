import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { API_BASE_URL, ApiError, apiRequest } from '@/lib/api'
import { getAuthToken } from '@/lib/auth-token'
import {
  apiErrorEnvelopeSchema,
  auditLogPageSchema,
  categoryPageSchema,
  type AuditLogPage,
  type CategoryPage,
} from '@/validations/schemas'

/** Kolom sort yang diizinkan backend untuk jejak audit. */
export const AUDIT_LOG_SORT_FIELDS = ['createdAt', 'aksi'] as const
export type AuditLogSortField = (typeof AUDIT_LOG_SORT_FIELDS)[number]

export type AuditLogSortOrder = 'asc' | 'desc'

export type AuditLogListParams = {
  page: number
  pageSize: number
  search: string
  aksi: string | null
  sort: AuditLogSortField
  order: AuditLogSortOrder
}

/** Satu halaman jejak audit dari GET /api/v1/audit-logs. */
export async function fetchAuditLogs(params: AuditLogListParams): Promise<AuditLogPage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort,
    order: params.order,
  })
  if (params.search) {
    query.set('search', params.search)
  }
  if (params.aksi) {
    query.set('aksi', params.aksi)
  }
  const payload = await apiRequest<unknown>(`/audit-logs?${query.toString()}`)
  return auditLogPageSchema.parse(payload)
}

export function useAuditLogs(params: AuditLogListParams) {
  return useQuery({
    queryKey: ['audit-logs', params] as const,
    queryFn: () => fetchAuditLogs(params),
    placeholderData: keepPreviousData,
    staleTime: 10_000,
    retry: false,
  })
}

/** Mengunduh jejak audit dalam cakupan pemohon sebagai berkas CSV. */
export async function exportAuditLogsCsv(filters: {
  aksi: string | null
  search: string
}) {
  const query = new URLSearchParams()
  if (filters.aksi) {
    query.set('aksi', filters.aksi)
  }
  if (filters.search) {
    query.set('search', filters.search)
  }
  const headers: Record<string, string> = {}
  const token = getAuthToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  const response = await fetch(`${API_BASE_URL}/audit-logs/export?${query.toString()}`, {
    headers,
  })
  if (!response.ok) {
    const body: unknown = await response.json().catch(() => undefined)
    const parsed = apiErrorEnvelopeSchema.safeParse(body)
    const code = parsed.success ? parsed.data.error.code : 'internal_error'
    const message = parsed.success ? (parsed.data.error.message ?? code) : code
    throw new ApiError({ status: response.status, code, message })
  }
  const disposition = response.headers.get('Content-Disposition') ?? ''
  const match = /filename="?([^"]+)"?/.exec(disposition)
  const filename = match?.[1] ?? 'audit-log.csv'
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

/** Kolom sort yang diizinkan backend untuk daftar kategori. */
export const CATEGORY_SORT_FIELDS = ['kode', 'nama', 'createdAt', 'updatedAt'] as const
export type CategorySortField = (typeof CATEGORY_SORT_FIELDS)[number]

export type CategorySortOrder = 'asc' | 'desc'

export type CategoryListParams = {
  page: number
  pageSize: number
  search: string
  sort: CategorySortField
  order: CategorySortOrder
}

/** Satu halaman kategori aktif dari GET /api/v1/categories. */
export async function fetchCategories(params: CategoryListParams): Promise<CategoryPage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort,
    order: params.order,
  })
  if (params.search) {
    query.set('search', params.search)
  }
  const payload = await apiRequest<unknown>(`/categories?${query.toString()}`)
  return categoryPageSchema.parse(payload)
}

export function useCategories(params: CategoryListParams) {
  return useQuery({
    queryKey: ['categories', params] as const,
    queryFn: () => fetchCategories(params),
    placeholderData: keepPreviousData,
    staleTime: 30_000,
    retry: false,
  })
}
