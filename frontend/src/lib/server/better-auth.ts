import { betterAuth } from 'better-auth'
import { createAuthMiddleware, isAPIError } from 'better-auth/api'
import { twoFactor } from 'better-auth/plugins'
import { tanstackStartCookies } from 'better-auth/tanstack-start'
import pg from 'pg'

/**
 * Runtime autentikasi Better Auth yang tertanam di server TanStack Start.
 * Memakai tabel "user", "session", "account", dan "verification" yang dibuat
 * migrasi backend; jangan membuat skema terpisah lewat better-auth CLI.
 *
 * Modul ini hanya untuk server. Jangan diimpor dari komponen klien.
 */
export class AuthNotConfiguredError extends Error {
  constructor() {
    super('DATABASE_URL dan BETTER_AUTH_SECRET wajib diisi untuk runtime autentikasi.')
    this.name = 'AuthNotConfiguredError'
  }
}

function isProduction(): boolean {
  return process.env.NODE_ENV === 'production' || process.env.APP_ENV === 'production'
}

const SIGN_IN_PATHS = new Set(['/sign-in/email', '/sign-in/social'])
const OAUTH_CALLBACK_PREFIXES = ['/callback/', '/oauth2/callback/']

// Kode kegagalan harus aman ditulis ke log: token pendek yang dikenali,
// bukan teks bebas (mis. error_description dari penyedia OAuth).
const REASON_PATTERN = /^[A-Za-z0-9_.-]{1,64}$/

function sanitizeReason(value: unknown): string | null {
  if (typeof value !== 'string') {
    return null
  }
  const trimmed = value.trim()
  return REASON_PATTERN.test(trimmed) ? trimmed : null
}

function redirectLocation(headers: unknown): string | null {
  if (!headers) {
    return null
  }
  if (typeof (headers as Headers).get === 'function') {
    return (headers as Headers).get('location')
  }
  if (typeof headers === 'object') {
    const location = (headers as Record<string, unknown>)['location']
    return typeof location === 'string' ? location : null
  }
  return null
}

/**
 * Menarik kode kegagalan dari respons Better Auth, atau `null` bila
 * permintaan berhasil. Tidak pernah memuat isi formulir, email, atau
 * kredensial. Kasus sukses OAuth adalah redirect 302 tanpa parameter
 * `error` pada URL tujuan.
 */
export function authFailureReason(returned: unknown): string | null {
  if (!isAPIError(returned)) {
    return null
  }
  const error = returned as {
    statusCode?: unknown
    body?: { code?: unknown } | undefined
    headers?: unknown
  }
  const statusCode = typeof error.statusCode === 'number' ? error.statusCode : 0
  const location = redirectLocation(error.headers)

  if (location) {
    try {
      const url = new URL(location, 'https://internal.invalid')
      if (url.searchParams.has('error')) {
        return sanitizeReason(url.searchParams.get('error')) ?? 'error_in_redirect'
      }
      // Redirect tanpa parameter error berarti login berhasil.
      return null
    } catch {
      return 'error_in_redirect'
    }
  }

  if (statusCode >= 400) {
    return sanitizeReason(error.body?.code) ?? `http_${statusCode}`
  }

  return null
}

type AuthLogContext = {
  path?: unknown
  context?: { returned?: unknown } | undefined
}

/**
 * Menulis satu baris JSON `{ event: 'auth_failed', ... }` ke stdout bila
 * konteks adalah kegagalan pada jalur masuk. Diekspor untuk pengujian.
 */
export function logAuthFailure(ctx: AuthLogContext): void {
  const path = typeof ctx.path === 'string' ? ctx.path : ''
  const watched =
    SIGN_IN_PATHS.has(path) ||
    OAUTH_CALLBACK_PREFIXES.some((prefix) => path.startsWith(prefix))
  if (!watched) {
    return
  }
  const reason = authFailureReason(ctx.context?.returned)
  if (!reason) {
    return
  }
  console.error(
    JSON.stringify({
      event: 'auth_failed',
      path,
      reason,
      ts: new Date().toISOString(),
    }),
  )
}

// hooks.after berjalan setelah penangan selesai, termasuk saat penangan
// melempar APIError, dan tidak boleh menggagalkan alur autentikasi.
const authFailureLogger = createAuthMiddleware(async (ctx) => {
  try {
    logAuthFailure(ctx)
  } catch {
    // Log observasi tidak boleh mematahkan login/logout.
  }
})

function createAuth() {
  const connectionString = process.env.DATABASE_URL?.trim()
  const secret = process.env.BETTER_AUTH_SECRET?.trim()
  if (!connectionString || !secret) {
    throw new AuthNotConfiguredError()
  }

  const pool = new pg.Pool({ connectionString, max: 5 })
  const trustedOrigins = (process.env.TRUSTED_ORIGINS ?? '')
    .split(',')
    .map((origin) => origin.trim())
    .filter((origin) => origin.length > 0)
  const googleClientId = process.env.GOOGLE_CLIENT_ID?.trim()
  const googleClientSecret = process.env.GOOGLE_CLIENT_SECRET?.trim()
  const socialProviders =
    googleClientId && googleClientSecret
      ? {
          google: {
            clientId: googleClientId,
            clientSecret: googleClientSecret,
          },
        }
      : undefined

  return betterAuth({
    baseURL: process.env.BETTER_AUTH_URL?.trim() || undefined,
    secret,
    // Pool pg langsung didukung adapter Kysely ("connect" in db). Jangan membungkus
    // dalam Kysely: instance Kysely tidak memaparkan properti "dialect" publik.
    database: pool,
    advanced: {
      // Tabel Better Auth (user/session/account/verification) memakai uuid PK
      // tanpa DEFAULT di basis data. Strategi bawaan "uuid" berasumsi DB yang
      // mengisinya, sehingga id ikut null dan insert gagal. UUID dibuat di sini.
      database: { generateId: () => crypto.randomUUID() },
      defaultCookieAttributes: {
        sameSite: 'lax',
        httpOnly: true,
        secure: isProduction(),
      },
      useSecureCookies: isProduction(),
    },
    session: {
      expiresIn: 60 * 60 * 24 * 7,
      updateAge: 60 * 60 * 24,
    },
    user: {
      fields: { name: 'nama' },
    },
    emailAndPassword: {
      enabled: true,
      // Tidak ada pendaftaran mandiri. Akun dibuat administrator lewat
      // scripts/provision-accounts.ts.
      disableSignUp: true,
      minPasswordLength: 8,
    },
    rateLimit: {
      enabled: true,
      window: 10,
      max: 100,
      storage: 'memory',
    },
    trustedOrigins: trustedOrigins.length > 0 ? trustedOrigins : undefined,
    socialProviders,
    hooks: {
      after: authFailureLogger,
    },
    plugins: [twoFactor({ issuer: 'SIMPEG' }), tanstackStartCookies()],
  })
}

let cached: ReturnType<typeof createAuth> | null = null

/** Runtime autentikasi bawa-satu-per-proses; dilempar ulang bila belum dikonfigurasi. */
export function getAuth(): ReturnType<typeof createAuth> {
  if (!cached) {
    cached = createAuth()
  }

  return cached
}
