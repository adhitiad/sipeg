import { afterEach, describe, expect, it, vi } from 'vitest'
import { API_BASE_URL, ApiError, apiRequest, isUnauthorized } from '@/lib/api'
import { setAuthToken } from '@/lib/auth-token'

function fakeResponse(status: number, payload: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => payload,
  } as unknown as Response
}

function stubFetch(response: Response | Error) {
  const fetchMock = vi.fn()
  if (response instanceof Error) {
    fetchMock.mockRejectedValue(response)
  } else {
    fetchMock.mockResolvedValue(response)
  }
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

async function captureError(promise: Promise<unknown>): Promise<ApiError> {
  try {
    await promise
  } catch (error) {
    return error as ApiError
  }

  throw new Error('permintaan ini seharusnya gagal')
}

describe('apiRequest', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    setAuthToken(null)
  })

  it('mengirim token sesi pada header Authorization', async () => {
    setAuthToken('token-uji')
    const fetchMock = stubFetch(fakeResponse(200, { userId: 'u-1' }))

    await apiRequest('/me')

    const [, options] = fetchMock.mock.calls[0]
    expect(options.headers.Authorization).toBe('Bearer token-uji')
  })

  it('memakai prefix versi /api/v1 pada alamat permintaan', async () => {
    const fetchMock = stubFetch(fakeResponse(200, {}))

    await apiRequest('/me')

    const [url] = fetchMock.mock.calls[0]
    expect(url).toBe(`${API_BASE_URL}/me`)
    expect(url).toContain('/api/v1')
  })

  it('tidak mengirim token pada permintaan tanpa autentikasi', async () => {
    setAuthToken('token-uji')
    const fetchMock = stubFetch(fakeResponse(200, {}))

    await apiRequest('/health', { authenticated: false })

    const [, options] = fetchMock.mock.calls[0]
    expect(options.headers.Authorization).toBeUndefined()
  })

  it('mengubah amplop error seragam menjadi ApiError', async () => {
    stubFetch(
      fakeResponse(403, {
        error: {
          code: 'forbidden',
          message: 'Anda tidak memiliki izin untuk tindakan ini.',
          requestId: 'req-123',
          fields: { role: 'wajib diisi' },
        },
      }),
    )

    const error = await captureError(apiRequest('/employees'))

    expect(error).toBeInstanceOf(ApiError)
    expect(error.status).toBe(403)
    expect(error.code).toBe('forbidden')
    expect(error.requestId).toBe('req-123')
    expect(error.fields).toEqual({ role: 'wajib diisi' })
    expect(error.message).toBe('Anda tidak memiliki izin untuk tindakan ini.')
  })

  it('memberi pesan Indonesia bila kode error tidak dikenal', async () => {
    stubFetch(fakeResponse(418, { error: { code: 'teapot' } }))

    const error = await captureError(apiRequest('/me'))

    expect(error).toBeInstanceOf(ApiError)
    expect(error.message).toContain('Permintaan gagal dengan status 418')
  })

  it('menandai kesalahan 401 sebagai unauthorized', async () => {
    stubFetch(fakeResponse(401, { error: { code: 'unauthorized' } }))

    const error = await captureError(apiRequest('/me'))

    expect(isUnauthorized(error)).toBe(true)
  })

  it('mengubah kegagalan jaringan menjadi ApiError', async () => {
    stubFetch(new TypeError('gagal koneksi'))

    const error = await captureError(apiRequest('/me'))

    expect(error).toBeInstanceOf(ApiError)
    expect(error.code).toBe('network_error')
    expect(error.message).toContain('Gagal menghubungi server SIMPEG')
  })

  it('mengembalikan void untuk respons 204', async () => {
    stubFetch(fakeResponse(204, undefined))

    await expect(apiRequest('/auth/logout', { method: 'POST' })).resolves.toBeUndefined()
  })
})
