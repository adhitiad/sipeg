import {
  DEFAULT_LOCALE,
  LOCALE_STORAGE_KEY,
  isLocale,
  type Locale,
} from '@/i18n/dictionary'

export const LOCALE_EVENT = 'simpeg:locale-change'

export function readStoredLocale(): Locale {
  try {
    const stored = window.localStorage.getItem(LOCALE_STORAGE_KEY)
    if (isLocale(stored)) {
      return stored
    }
  } catch {
    // localStorage tidak tersedia (mis. mode privat); bahasa bawaan dipakai.
  }

  return DEFAULT_LOCALE
}

export function readServerLocale(): Locale {
  return DEFAULT_LOCALE
}

export function subscribeToLocale(callback: () => void): () => void {
  window.addEventListener('storage', callback)
  window.addEventListener(LOCALE_EVENT, callback)

  return () => {
    window.removeEventListener('storage', callback)
    window.removeEventListener(LOCALE_EVENT, callback)
  }
}

/** Menulis bahasa baru dan memberi tahu seluruh pelanggan di tab ini. */
export function setStoredLocale(next: Locale) {
  try {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, next)
  } catch {
    // Penyimpanan gagal; bahasa hanya berlaku untuk sesi tab ini.
  }

  window.dispatchEvent(new Event(LOCALE_EVENT))
}
