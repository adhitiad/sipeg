import type { Locale, MessageKey, MessageParams } from '@/i18n/dictionary'

export interface LocaleContextValue {
  locale: Locale
  setLocale: (locale: Locale) => void
  t: (key: MessageKey, params?: MessageParams) => string
}

export type Translate = (key: MessageKey) => string
