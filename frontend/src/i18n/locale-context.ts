import { createContext, useContext } from 'react'
import type { LocaleContextValue } from '@/lib/types/i18n'

export const LocaleContext = createContext<LocaleContextValue | null>(null)

export function useLocale(): LocaleContextValue {
  const context = useContext(LocaleContext)
  if (!context) {
    throw new Error('useLocale harus dipakai di dalam LocaleProvider')
  }

  return context
}

export function useMessages() {
  return useLocale().t
}
