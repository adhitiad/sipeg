import { createStore } from '@tanstack/react-store'
import type { AuthTokenState } from '@/lib/types/auth'

/**
 * Token sesi hanya hidup di memori tab browser. Token tidak pernah ditulis ke
 * localStorage atau sessionStorage karena PRD §11 melarang menyimpan token
 * sesi di penyimpanan browser.
 */
export const authSessionStore = createStore<AuthTokenState>({ token: null })

export function getAuthToken(): string | null {
  return authSessionStore.get().token
}

export function setAuthToken(token: string | null): void {
  authSessionStore.setState(() => ({ token }))
}
