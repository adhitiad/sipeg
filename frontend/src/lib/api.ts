import { getAuthToken } from '@/lib/auth-token'
import { apiErrorEnvelopeSchema } from '@/validations/schemas'
import type { ApiErrorInit, ApiRequestOptions } from '@/lib/types/api'

/**
 * Basis URL API backend. Seluruh panggilan memakai prefix versi /api/v1;
 * path tanpa versi ditolak backend dengan 400 unversioned_path.
 */
export const API_BASE_URL = (
  import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'
).replace(/\/+$/, '')

const CODE_MESSAGES: Record<string, string> = {
  unauthorized: 'Sesi tidak valid atau sudah berakhir. Silakan masuk kembali.',
  forbidden: 'Anda tidak memiliki izin untuk tindakan ini.',
  not_found: 'Data yang diminta tidak ditemukan.',
  method_not_allowed: 'Metode HTTP tidak diizinkan untuk endpoint ini.',
  bad_request: 'Permintaan yang dikirim tidak valid.',
  invalid_query: 'Parameter pencarian tidak valid.',
  invalid_json: 'Format JSON pada permintaan tidak valid.',
  invalid_id: 'Pengenal pada permintaan tidak valid.',
  validation_error: 'Data yang dikirim belum lengkap atau belum sesuai aturan.',
  conflict: 'Data dengan nilai tersebut sudah ada.',
  not_ready: 'Layanan belum siap menerima permintaan.',
  unversioned_path: 'Path API harus memakai versi /api/v1.',
  service_unavailable: 'Layanan sedang tidak tersedia. Coba lagi beberapa saat lagi.',
  internal_error: 'Terjadi kesalahan pada server. Coba lagi beberapa saat lagi.',
}

/** Kesalahan API yang membawa bentuk error seragam backend. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly requestId: string | undefined
  readonly fields: Record<string, string> | undefined
  readonly rows: unknown

  constructor(init: ApiErrorInit) {
    super(init.message, { cause: init.cause })
    this.name = 'ApiError'
    this.status = init.status
    this.code = init.code
    this.requestId = init.requestId
    this.fields = init.fields
    this.rows = init.rows
  }
}

export function isUnauthorized(error: unknown): boolean {
  return error instanceof ApiError && error.status === 401
}

function resolveMessage(code: string, fallback: string): string {
  return CODE_MESSAGES[code] ?? fallback
}

/**
 * Melakukan permintaan ke backend SIMPEG.
 *
 * Sesi dibawa lewat header `Authorization: Bearer <token>` sesuai skema
 * `bearerSession` pada docs/openapi.yaml. Cookie tidak disertakan karena
 * middleware CORS backend hanya mengirim `Access-Control-Allow-Origin` tanpa
 * `Access-Control-Allow-Credentials`, sehingga request lintas origin dengan
 * cookie akan diblokir browser.
 */
export async function apiRequest<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<T> {
  const { method = 'GET', body, signal, authenticated = true } = options

  const headers: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  if (authenticated) {
    const token = getAuthToken()
    if (token) {
      headers.Authorization = `Bearer ${token}`
    }
  }

  let response: Response
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method,
      headers,
      credentials: 'omit',
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    })
  } catch (cause) {
    throw new ApiError({
      status: 0,
      code: 'network_error',
      message:
        'Gagal menghubungi server SIMPEG. Periksa koneksi jaringan Anda lalu coba lagi.',
      cause,
    })
  }

  if (response.status === 204) {
    return undefined as T
  }

  const payload = await response.json().catch(() => null)

  if (!response.ok) {
    const parsed = apiErrorEnvelopeSchema.safeParse(payload)
    const detail = parsed.success ? parsed.data.error : null
    const code = detail?.code ?? 'unknown_error'

    throw new ApiError({
      status: response.status,
      code,
      message: resolveMessage(
        code,
        detail?.message ??
          `Permintaan gagal dengan status ${response.status}. Coba lagi beberapa saat lagi.`,
      ),
      requestId: detail?.requestId,
      fields: detail?.fields,
      rows: detail?.rows,
    })
  }

  return payload as T
}
