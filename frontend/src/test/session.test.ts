import { afterEach, describe, expect, it, vi } from 'vitest'
import { setAuthToken } from '@/lib/auth-token'
import { fetchCurrentAccount } from '@/lib/session'

const ACCOUNT = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'pegawai@instansi.go.id',
  roleName: 'operator',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
}

function fakeResponse(status: number, payload: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => payload,
  } as unknown as Response
}

function stubFetch(response: Response) {
  const fetchMock = vi.fn().mockResolvedValue(response)
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('fetchCurrentAccount', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    setAuthToken(null)
  })

  it('mengembalikan identitas principal dari backend', async () => {
    setAuthToken('token-uji')
    stubFetch(fakeResponse(200, ACCOUNT))

    await expect(fetchCurrentAccount()).resolves.toEqual(ACCOUNT)
  })

  it('mengembalikan null bila sesi tidak valid', async () => {
    setAuthToken('token-uji')
    stubFetch(fakeResponse(401, { error: { code: 'unauthorized' } }))

    await expect(fetchCurrentAccount()).resolves.toBeNull()
  })

  it('melempar kesalahan bila backend gagal lain', async () => {
    setAuthToken('token-uji')
    stubFetch(fakeResponse(503, { error: { code: 'service_unavailable' } }))

    await expect(fetchCurrentAccount()).rejects.toMatchObject({
      status: 503,
      code: 'service_unavailable',
    })
  })

  it('menolak respons yang tidak sesuai kontrak', async () => {
    setAuthToken('token-uji')
    stubFetch(fakeResponse(200, { userId: 'bukan-uuid' }))

    await expect(fetchCurrentAccount()).rejects.toBeTruthy()
  })

  it('tidak mengirim permintaan bila tidak ada token di memori', async () => {
    const fetchMock = stubFetch(fakeResponse(200, ACCOUNT))

    await expect(fetchCurrentAccount()).resolves.toBeNull()
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
