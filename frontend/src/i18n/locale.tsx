import {
  useCallback,
  useEffect,
  useMemo,
  useSyncExternalStore,
  type ReactNode,
} from 'react'
import { translate, type Locale } from '@/i18n/dictionary'
import { LocaleContext } from '@/i18n/locale-context'
import type { LocaleContextValue } from '@/lib/types/i18n'
import {
  readServerLocale,
  readStoredLocale,
  setStoredLocale,
  subscribeToLocale,
} from '@/i18n/locale-store'

/**
 * Bahasa bawaan server dan render hidrasi pertama selalu Bahasa Indonesia.
 * Pilihan yang tersimpan di localStorage diterapkan lewat useSyncExternalStore
 * tanpa ketidakcocokan hidrasi.
 */
export function LocaleProvider({ children }: { children: ReactNode }) {
  const locale = useSyncExternalStore(
    subscribeToLocale,
    readStoredLocale,
    readServerLocale,
  )

  useEffect(() => {
    document.documentElement.lang = locale === 'id' ? 'id' : 'en'
  }, [locale])

  const setLocale = useCallback((next: Locale) => {
    setStoredLocale(next)
  }, [])

  const value = useMemo<LocaleContextValue>(
    () => ({
      locale,
      setLocale,
      t: (key, params) => translate(locale, key, params),
    }),
    [locale, setLocale],
  )

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
}
