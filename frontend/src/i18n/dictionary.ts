import { uiMessagesEn } from './messages/ui.en'
import { uiMessagesId } from './messages/ui.id'
import { contentMessagesEn } from './messages/content.en'
import { contentMessagesId } from './messages/content.id'

/**
 * Gabungan seluruh pesan per bahasa. Kunci diverifikasi lewat `satisfies` pada
 * berkas pesan; di sini hanya penggabungan yang dilakukan.
 */
export const messages = {
  id: { ...uiMessagesId, ...contentMessagesId },
  en: { ...uiMessagesEn, ...contentMessagesEn },
} as const

export type Locale = keyof typeof messages
export type MessageKey = keyof (typeof messages)[Locale]
export type MessageParams = Record<string, string>

export const DEFAULT_LOCALE: Locale = 'id'
export const SUPPORTED_LOCALES: readonly Locale[] = ['id', 'en']
export const LOCALE_STORAGE_KEY = 'simpeg:locale'

export function isLocale(value: string | null): value is Locale {
  return value === 'id' || value === 'en'
}

export function translate(
  locale: Locale,
  key: MessageKey,
  params?: MessageParams,
): string {
  const template = messages[locale][key] ?? key
  if (!params) {
    return template
  }

  return Object.entries(params).reduce(
    (text, [name, value]) => text.replaceAll(`{${name}}`, value),
    template,
  )
}
