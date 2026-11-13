import { Moon, Sun } from 'lucide-react'
import { useEffect } from 'react'
import { useMessages } from '@/i18n/locale-context'
import { applyThemeMode, setThemeMode, useThemeMode } from '@/lib/theme'

export default function ThemeToggle() {
  const mode = useThemeMode()
  const t = useMessages()

  useEffect(() => {
    applyThemeMode(mode)
  }, [mode])

  const isDark = mode === 'dark'

  return (
    <button
      type="button"
      onClick={() => setThemeMode(isDark ? 'light' : 'dark')}
      aria-label={isDark ? t('theme.switchToLight') : t('theme.switchToDark')}
      title={isDark ? t('theme.switchToLight') : t('theme.switchToDark')}
      className="inline-flex h-11 w-11 items-center justify-center rounded-full border border-[var(--chip-line)] bg-[var(--chip-bg)] text-[var(--sea-ink)] transition hover:bg-[var(--link-bg-hover)] sm:h-9 sm:w-9"
    >
      {isDark ? (
        <Sun className="h-4 w-4" aria-hidden="true" />
      ) : (
        <Moon className="h-4 w-4" aria-hidden="true" />
      )}
    </button>
  )
}
