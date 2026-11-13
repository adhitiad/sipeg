import { afterEach, describe, expect, it, vi } from 'vitest'
import { authFailureReason, logAuthFailure } from '@/lib/server/better-auth'

function apiError(init: { statusCode: number; code?: string; location?: string }) {
  const error = new Error('api error') as Error & {
    name: string
    statusCode: number
    status: string
    body?: { code: string }
    headers: Headers
  }
  error.name = 'APIError'
  error.status = String(init.statusCode)
  error.statusCode = init.statusCode
  if (init.code !== undefined) {
    error.body = { code: init.code }
  }
  const headers = new Headers()
  if (init.location) {
    headers.set('location', init.location)
  }
  error.headers = headers
  return error
}

describe('authFailureReason', () => {
  it('mengambil kode dari body pada galat 4xx jalur masuk email', () => {
    const reason = authFailureReason(
      apiError({ statusCode: 401, code: 'INVALID_EMAIL_OR_PASSWORD' }),
    )
    expect(reason).toBe('INVALID_EMAIL_OR_PASSWORD')
  })

  it('mengambil kode error dari parameter URL pada redirect gagal OAuth', () => {
    const reason = authFailureReason(
      apiError({
        statusCode: 302,
        location:
          'https://simpeg.example.test/api/auth/error?error=state_not_found&error_description=State%20not%20found',
      }),
    )
    expect(reason).toBe('state_not_found')
  })

  it('tidak menganggap redirect sukses login sebagai kegagalan', () => {
    const reason = authFailureReason(
      apiError({
        statusCode: 302,
        location: 'https://simpeg.example.test/workspace',
      }),
    )
    expect(reason).toBeNull()
  })

  it('mengganti kode yang tidak aman dengan penanda generik', () => {
    const reason = authFailureReason(
      apiError({
        statusCode: 401,
        code: 'bukan kode; DROP TABLE user',
      }),
    )
    expect(reason).toBe('http_401')
  })

  it('mengganti error pada redirect yang tidak aman dengan penanda generik', () => {
    const reason = authFailureReason(
      apiError({
        statusCode: 302,
        location:
          'https://simpeg.example.test/api/auth/error?error=' +
          encodeURIComponent('aku<script>alert(1)</script>'),
      }),
    )
    expect(reason).toBe('error_in_redirect')
  })

  it('mengabaikan respons sukses (bukan APIError)', () => {
    expect(authFailureReason({ token: 'abc', user: {} })).toBeNull()
    expect(authFailureReason(undefined)).toBeNull()
    expect(authFailureReason('string acak')).toBeNull()
  })
})

describe('logAuthFailure', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('menulis satu baris JSON auth_failed tanpa kredensial', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})

    logAuthFailure({
      path: '/sign-in/email',
      context: {
        returned: apiError({
          statusCode: 401,
          code: 'INVALID_EMAIL_OR_PASSWORD',
        }),
      },
    })

    expect(spy).toHaveBeenCalledTimes(1)
    const line = spy.mock.calls[0]?.[0] as string
    const parsed = JSON.parse(line) as Record<string, unknown>
    expect(parsed['event']).toBe('auth_failed')
    expect(parsed['path']).toBe('/sign-in/email')
    expect(parsed['reason']).toBe('INVALID_EMAIL_OR_PASSWORD')
    expect(line).not.toContain('password')
    expect(line).not.toContain('kata-sandi')
  })

  it('mencatat kegagalan callback OAuth dengan kode dari URL redirect', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})

    logAuthFailure({
      path: '/callback/google',
      context: {
        returned: apiError({
          statusCode: 302,
          location: 'https://simpeg.example.test/api/auth/error?error=state_not_found',
        }),
      },
    })

    const line = spy.mock.calls[0]?.[0] as string
    const parsed = JSON.parse(line) as Record<string, unknown>
    expect(parsed['reason']).toBe('state_not_found')
    expect(parsed['path']).toBe('/callback/google')
  })

  it('tidak menulis apa-apa untuk login sukses', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})

    logAuthFailure({
      path: '/sign-in/email',
      context: { returned: { token: 'abc', user: { id: 'u' } } },
    })

    expect(spy).not.toHaveBeenCalled()
  })

  it('mengabaikan jalur yang tidak relevan', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})

    logAuthFailure({
      path: '/get-session',
      context: { returned: apiError({ statusCode: 401, code: 'UNAUTHORIZED' }) },
    })

    expect(spy).not.toHaveBeenCalled()
  })

  it('tidak pernah melempar error meski konteks rusak', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})

    expect(() =>
      logAuthFailure({
        path: '/sign-in/email',
        context: { returned: apiError({ statusCode: 500 }) },
      }),
    ).not.toThrow()
    expect(spy).toHaveBeenCalledTimes(1)
  })
})
