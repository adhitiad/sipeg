import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { z } from 'zod'
import { ApiError, apiRequest } from '@/lib/api'
import { getAuthToken, setAuthToken } from '@/lib/auth-token'
import {
  authErrorSchema,
  currentAccountSchema,
  enableTwoFactorResponseSchema,
  signInResponseSchema,
  twoFactorChallengeSchema,
  twoFactorVerificationSchema,
  type ChangePasswordInput,
  type CurrentAccount,
  type EnableTwoFactorResponse,
  type UpdateProfileInput,
} from '@/validations/schemas'

/**
 * Basis URL runtime autentikasi Better Auth. Bawaan adalah runtime yang
 * tertanam di server TanStack Start pada jalur yang sama (`/api/auth`). Setel
 * VITE_AUTH_BASE_URL hanya bila runtime berjalan di origin terpisah.
 */
export const AUTH_BASE_URL = (import.meta.env.VITE_AUTH_BASE_URL ?? '/api/auth').replace(
  /\/+$/,
  '',
)

export const sessionQueryKey = ['session'] as const

export function isAuthRuntimeConfigured(): boolean {
  return AUTH_BASE_URL.length > 0
}

export class AuthRuntimeError extends Error {
  readonly status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'AuthRuntimeError'
    this.status = status
  }
}

/**
 * Membaca identitas principal dari backend. Mengembalikan `null` bila sesi
 * tidak valid (401); error lain dilempar agar UI bisa menampilkan kegagalan.
 * Bila tidak ada token di memori (mis. setelah muat ulang), permintaan tidak
 * dikirim sama sekali: hasilnya pasti 401 dan hanya membisingkan konsol.
 */
export async function fetchCurrentAccount(): Promise<CurrentAccount | null> {
  if (!getAuthToken()) {
    return null
  }

  try {
    const payload = await apiRequest<unknown>('/me')
    return currentAccountSchema.parse(payload)
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      return null
    }
    throw error
  }
}

export function useSession() {
  return useQuery({
    queryKey: sessionQueryKey,
    queryFn: fetchCurrentAccount,
    retry: false,
    staleTime: 30_000,
  })
}

async function requestSignIn(email: string, password: string): Promise<SignInOutcome> {
  if (!isAuthRuntimeConfigured()) {
    throw new AuthRuntimeError(
      'Runtime autentikasi belum dikonfigurasi. Setel VITE_AUTH_BASE_URL lalu jalankan ulang server frontend.',
      0,
    )
  }

  let response: Response
  try {
    response = await fetch(`${AUTH_BASE_URL}/sign-in/email`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ email, password }),
    })
  } catch {
    throw new AuthRuntimeError(
      'Gagal menghubungi runtime autentikasi. Periksa konfigurasi VITE_AUTH_BASE_URL.',
      0,
    )
  }

  if (!response.ok) {
    const payload = await response.json().catch(() => null)
    const parsed = authErrorSchema.safeParse(payload)
    const fallback =
      response.status >= 500
        ? 'Layanan autentikasi mengalami gangguan. Coba lagi beberapa saat.'
        : 'Email atau kata sandi tidak dikenali. Periksa kembali data masuk Anda.'
    throw new AuthRuntimeError(
      parsed.success && parsed.data.message ? parsed.data.message : fallback,
      response.status,
    )
  }

  const payload = await response.json().catch(() => null)
  // Akun dengan 2FA aktif tidak mendapat sesi; runtime mengembalikan
  // tantangan {twoFactorRedirect: true} dan pengguna harus memasukkan kode.
  const challenge = twoFactorChallengeSchema.safeParse(payload)
  if (challenge.success) {
    return 'twoFactor'
  }
  const parsed = signInResponseSchema.safeParse(payload)
  if (!parsed.success) {
    throw new AuthRuntimeError(
      'Respons runtime autentikasi tidak sesuai kontrak. Silakan hubungi administrator.',
      response.status,
    )
  }

  setAuthToken(parsed.data.token)
  return 'session'
}

export type SignInOutcome = 'session' | 'twoFactor'

export type { EnableTwoFactorResponse }

export function useSignIn() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: { email: string; password: string }) =>
      requestSignIn(input.email, input.password),
    onSuccess: async (outcome) => {
      if (outcome === 'session') {
        await queryClient.invalidateQueries({ queryKey: sessionQueryKey })
      }
    },
  })
}

/** Mulai alur masuk Google: runtime mengembalikan URL otorisasi. */
export async function signInWithGoogle(): Promise<void> {
  let response: Response
  try {
    response = await fetch(`${AUTH_BASE_URL}/sign-in/social`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ provider: 'google', callbackURL: '/workspace' }),
    })
  } catch {
    throw new AuthRuntimeError(
      'Gagal menghubungi runtime autentikasi. Periksa konfigurasi VITE_AUTH_BASE_URL.',
      0,
    )
  }

  const payload: unknown = await response.json().catch(() => null)
  const parsed = z
    .object({ url: z.url().startsWith('https://'), redirect: z.boolean().optional() })
    .safeParse(payload)
  if (!response.ok || !parsed.success) {
    const code =
      payload && typeof payload === 'object' && 'code' in payload
        ? (payload as { code?: unknown }).code
        : undefined
    throw new AuthRuntimeError(
      code === 'PROVIDER_NOT_FOUND'
        ? 'Masuk dengan Google belum dikonfigurasi oleh administrator.'
        : 'Google tidak menerima permintaan masuk Anda. Coba lagi.',
      response.status,
    )
  }
  window.location.assign(parsed.data.url)
}

/** Aktifkan TOTP untuk akun yang sedang masuk. */
export async function enableTwoFactor(
  password: string,
): Promise<EnableTwoFactorResponse> {
  const response = await fetch(`${AUTH_BASE_URL}/two-factor/enable`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify({ password, method: 'totp' }),
  })
  const payload: unknown = await response.json().catch(() => null)
  if (!response.ok) {
    throw new AuthRuntimeError(
      authErrorMessage(payload) ?? 'Gagal mengaktifkan 2FA. Periksa kata sandi Anda.',
      response.status,
    )
  }
  const parsed = enableTwoFactorResponseSchema.safeParse(payload)
  if (!parsed.success) {
    throw new AuthRuntimeError(
      'Respons runtime tidak sesuai kontrak (enable 2FA).',
      response.status,
    )
  }
  return parsed.data
}

/** Nonaktifkan TOTP untuk akun yang sedang masuk. */
export async function disableTwoFactor(password: string): Promise<void> {
  const response = await fetch(`${AUTH_BASE_URL}/two-factor/disable`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify({ password }),
  })
  if (!response.ok) {
    const payload: unknown = await response.json().catch(() => null)
    throw new AuthRuntimeError(
      authErrorMessage(payload) ?? 'Gagal menonaktifkan 2FA. Periksa kata sandi Anda.',
      response.status,
    )
  }
}

/** Verifikasi kode TOTP; menyelesaikan tantangan 2FA saat masuk. */
export async function verifyTotp(code: string): Promise<void> {
  const response = await fetch(`${AUTH_BASE_URL}/two-factor/verify-totp`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify({ code }),
  })
  const payload: unknown = await response.json().catch(() => null)
  if (!response.ok) {
    throw new AuthRuntimeError(
      'Kode verifikasi tidak cocok. Periksa aplikasi autentikator Anda.',
      response.status,
    )
  }
  const parsed = twoFactorVerificationSchema.safeParse(payload)
  if (!parsed.success) {
    throw new AuthRuntimeError(
      'Respons runtime tidak sesuai kontrak (verify 2FA).',
      response.status,
    )
  }
  setAuthToken(parsed.data.token)
}

/** Verifikasi kode cadangan; menyelesaikan tantangan 2FA saat masuk. */
export async function verifyBackupCode(code: string): Promise<void> {
  const response = await fetch(`${AUTH_BASE_URL}/two-factor/verify-backup-code`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify({ code }),
  })
  const payload: unknown = await response.json().catch(() => null)
  if (!response.ok) {
    throw new AuthRuntimeError(
      'Kode cadangan tidak cocok atau sudah dipakai.',
      response.status,
    )
  }
  const parsed = twoFactorVerificationSchema.safeParse(payload)
  if (!parsed.success) {
    throw new AuthRuntimeError(
      'Respons runtime tidak sesuai kontrak (backup 2FA).',
      response.status,
    )
  }
  setAuthToken(parsed.data.token)
}

function authErrorMessage(payload: unknown): string | undefined {
  const parsed = authErrorSchema.safeParse(payload)
  if (!parsed.success) {
    return undefined
  }
  const message = parsed.data.message
  return typeof message === 'string' && message.length > 0 ? message : undefined
}

async function requestSignOut(): Promise<void> {
  try {
    await apiRequest<void>('/auth/logout', { method: 'POST' })
  } catch {
    // Pencabutan sesi di backend bersifat idempoten; kegagalan tidak
    // menghalangi pengguna keluar dari antarmuka.
  }

  if (isAuthRuntimeConfigured()) {
    try {
      // Endpoint sign-out Better Auth memvalidasi content-type dan body JSON;
      // kirim objek kosong karena tidak ada parameter opsional yang dipakai.
      await fetch(`${AUTH_BASE_URL}/sign-out`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: '{}',
      })
    } catch {
      // Token JWT dicabut oleh runtime autentikasi. Bila runtime tidak
      // terjangkau, token lokal tetap dibuang di bawah.
    }
  }

  setAuthToken(null)
}

export function useSignOut() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: requestSignOut,
    onSuccess: async () => {
      queryClient.clear()
      await queryClient.invalidateQueries({ queryKey: sessionQueryKey })
    },
  })
}

/** Perbarui profil mandiri (nama dan/atau nomor HP) lewat PATCH /api/v1/me. */
export async function updateCurrentAccount(
  input: UpdateProfileInput,
): Promise<CurrentAccount> {
  const payload = await apiRequest<unknown>('/me', { method: 'PATCH', body: input })
  return currentAccountSchema.parse(payload)
}

export function useUpdateProfile() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: updateCurrentAccount,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: sessionQueryKey })
    },
  })
}

/**
 * Ubah kata sandi lewat runtime Better Auth yang tertanam. Semua sesi lain
 * dicabut; sesi saat ini tetap berlaku.
 */
export async function changePassword(input: ChangePasswordInput): Promise<void> {
  const { currentPassword, newPassword } = input
  let response: Response
  try {
    response = await fetch(`${AUTH_BASE_URL}/change-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ currentPassword, newPassword, revokeOtherSessions: true }),
    })
  } catch {
    throw new AuthRuntimeError(
      'Gagal menghubungi runtime autentikasi. Periksa koneksi Anda.',
      0,
    )
  }

  if (!response.ok) {
    const payload: unknown = await response.json().catch(() => null)
    throw new AuthRuntimeError(
      authErrorMessage(payload) ??
        'Kata sandi tidak dapat diubah. Periksa kata sandi saat ini.',
      response.status,
    )
  }
}
