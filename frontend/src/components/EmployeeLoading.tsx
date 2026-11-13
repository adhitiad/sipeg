import { useMessages } from '@/i18n/locale-context'

export default function EmployeeLoading() {
  const t = useMessages()

  return (
    <div
      role="status"
      aria-live="polite"
      aria-busy="true"
      className="inline-flex items-center justify-center rounded-xl border border-[var(--line)] bg-[var(--surface)] p-4 text-[var(--sea-ink)] shadow-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ring)]"
    >
      <div
        className="relative flex items-center justify-center overflow-hidden rounded-xl border border-[var(--line)] bg-[var(--card)]"
        style={{ width: '10rem', height: '10rem' }}
      >
        <div className="absolute inset-0 rounded-full border border-[var(--line)]" />
        <div className="absolute inset-[1rem] rounded-full border border-dashed border-[var(--line)]" />
        <div
          className="absolute animate-spin"
          style={{ width: '7.5rem', height: '7.5rem', animationDuration: '2.8s' }}
        >
          <div
            className="absolute left-1/2 top-1 -translate-x-1/2 rounded-full bg-[var(--lagoon-deep)]"
            style={{ width: '1rem', height: '2rem' }}
          >
            <div
              className="mx-auto bg-[var(--lagoon)]"
              style={{ width: '0.0625rem', height: '1rem' }}
            />
          </div>
        </div>
        <div
          className="relative flex items-center justify-center rounded-full border border-[var(--line)] bg-[var(--surface)] shadow-inner"
          style={{ width: '3.5rem', height: '3.5rem' }}
        >
          <div
            className="absolute bg-[var(--line)]"
            style={{ width: '2rem', height: '0.0625rem' }}
          />
          <div
            className="absolute bg-[var(--line)]"
            style={{ width: '0.0625rem', height: '2rem' }}
          />
          <div
            className="rounded-full bg-[var(--sea-ink)]"
            style={{ width: '1rem', height: '1rem' }}
          />
        </div>
        <span className="absolute bottom-1 text-[var(--sea-ink-soft)] text-xs uppercase tracking-widest">
          {t('employee.loading.title')}
        </span>
      </div>
      <span className="sr-only">{t('employee.loading.description')}</span>
    </div>
  )
}
