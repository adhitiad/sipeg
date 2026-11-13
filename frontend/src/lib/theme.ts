import { useSyncExternalStore } from 'react'
import type { ThemeMode } from '@/lib/types/theme'

export const THEME_STORAGE_KEY = 'theme'
export const THEME_EVENT = 'simpeg:theme-change'

export const DEFAULT_THEME: ThemeMode = 'light'

export function isThemeMode(value: string | null): value is ThemeMode {
  return value === 'light' || value === 'dark'
}

export function readStoredMode(): ThemeMode {
  try {
    const stored = window.localStorage.getItem(THEME_STORAGE_KEY)
    if (isThemeMode(stored)) {
      return stored
    }
  } catch {
    // localStorage tidak tersedia; tema bawaan dipakai.
  }

  return DEFAULT_THEME
}

export function readServerMode(): ThemeMode {
  return DEFAULT_THEME
}

export function subscribeToTheme(callback: () => void): () => void {
  window.addEventListener('storage', callback)
  window.addEventListener(THEME_EVENT, callback)

  return () => {
    window.removeEventListener('storage', callback)
    window.removeEventListener(THEME_EVENT, callback)
  }
}

export function applyThemeMode(mode: ThemeMode) {
  const root = document.documentElement
  root.classList.remove('light', 'dark')
  root.classList.add(mode)
  root.setAttribute('data-theme', mode)
  root.style.colorScheme = mode
}

/** Menulis tema baru dan memberi tahu seluruh pelanggan di tab ini. */
export function setThemeMode(mode: ThemeMode) {
  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, mode)
  } catch {
    // Penyimpanan gagal; tema hanya berlaku untuk sesi tab ini.
  }

  applyThemeMode(mode)
  window.dispatchEvent(new Event(THEME_EVENT))
}

/** Mode tema yang berlaku saat ini; aman dipakai pada SSR. */
export function useThemeMode(): ThemeMode {
  return useSyncExternalStore(subscribeToTheme, readStoredMode, readServerMode)
}
