type LoadingRadarSize = 'sm' | 'md' | 'lg'

type LoadingRadarProps = {
  /** Ukuran radar. Ukuran `md` dipakai pada halaman daftar pegawai. */
  size?: LoadingRadarSize
  /** Label yang dibacakan pembaca layar. */
  label?: string
  /** Setel true bila teks di dekat indikator sudah menjelaskan keadaan. */
  decorative?: boolean
  className?: string
}

const FONT_SIZE: Record<LoadingRadarSize, string> = {
  sm: '0.5rem',
  md: '0.625rem',
  lg: '0.75rem',
}

/**
 * Indikator memuat berbentuk radar: satu lengan berputar menyapu cincin
 * konsentris untuk menunjukkan daftar sedang disaring. Bentuknya dipakai pada
 * halaman manajemen pegawai karena penyaringan unit (biro, fakultas, jurusan,
 * rektorat) memang proses penyaringan yang berjalan.
 *
 * Warna memakai token produk: lengan memakai lagoon, titik sorot memakai
 * palm. Gerak berhenti saat pengguna meminta `prefers-reduced-motion`, tetapi
 * bentuk radar tetap terbaca sebagai tanda data sedang dimuat.
 */
export default function LoadingRadar({
  size = 'md',
  label = 'Memuat',
  decorative = false,
  className = '',
}: LoadingRadarProps) {
  return (
    <span
      role={decorative ? undefined : 'status'}
      aria-live={decorative ? undefined : 'polite'}
      aria-busy={decorative ? undefined : 'true'}
      aria-label={decorative ? undefined : label}
      aria-hidden={decorative ? 'true' : undefined}
      className={`inline-flex flex-col items-center gap-3 ${className}`}
    >
      <span
        aria-hidden="true"
        style={{ fontSize: FONT_SIZE[size], width: '12em', height: '12em' }}
        className="relative flex items-center justify-center overflow-hidden rounded-full border border-[var(--line)] bg-[var(--surface-strong)]"
        // 12em mengikuti ukuran radar asli Uiverse; seluruh bagian di dalam
        // memakai em agar proporsinya tetap sama pada tiga ukuran.
      >
        <span
          className="absolute rounded-full border border-[var(--line)]"
          style={{ width: '9.5em', height: '9.5em' }}
        />
        <span
          className="absolute rounded-full border border-dashed border-[var(--line)]"
          style={{ width: '6.75em', height: '6.75em' }}
        />

        <span
          className="absolute rounded-full bg-[color-mix(in_oklab,var(--sea-ink)_18%,transparent)]"
          style={{ left: '1em', top: '1em', width: '0.35em', height: '0.35em' }}
        />
        <span
          className="absolute rounded-full bg-[color-mix(in_oklab,var(--sea-ink)_18%,transparent)]"
          style={{ right: '1em', top: '1em', width: '0.35em', height: '0.35em' }}
        />
        <span
          className="absolute rounded-full bg-[color-mix(in_oklab,var(--sea-ink)_18%,transparent)]"
          style={{ left: '1em', bottom: '1em', width: '0.35em', height: '0.35em' }}
        />
        <span
          className="absolute rounded-full bg-[color-mix(in_oklab,var(--sea-ink)_18%,transparent)]"
          style={{ right: '1em', bottom: '1em', width: '0.35em', height: '0.35em' }}
        />

        <span
          className="absolute bg-[color-mix(in_oklab,var(--sea-ink)_14%,transparent)]"
          style={{ width: '0.0625em', height: '7em' }}
        />
        <span
          className="absolute bg-[color-mix(in_oklab,var(--sea-ink)_14%,transparent)]"
          style={{ width: '7em', height: '0.0625em' }}
        />

        <span
          className="absolute motion-safe:animate-[spin_2.8s_linear_infinite]"
          style={{ width: '9.5em', height: '9.5em' }}
        >
          <span
            className="absolute left-1/2 top-1 -translate-x-1/2 rounded-full bg-[var(--sea-ink)]"
            style={{ width: '1.25em', height: '2.5em' }}
          >
            <span
              className="mx-auto bg-[var(--lagoon)]"
              style={{ marginTop: '0.75em', width: '0.0625em', height: '1em' }}
            />
          </span>
        </span>

        <span
          className="absolute rotate-45"
          style={{ width: '6.75em', height: '6.75em' }}
        >
          <span className="relative h-full w-full motion-safe:animate-[spin_4.25s_linear_reverse_infinite]">
            <span
              className="absolute left-0 top-1/2 rounded-full bg-[var(--palm)]"
              style={{
                width: '0.75em',
                height: '0.75em',
                transform: 'translateY(-50%)',
              }}
            />
            <span
              className="absolute right-0 top-1/2 rounded-full bg-[color-mix(in_oklab,var(--sea-ink)_18%,transparent)]"
              style={{
                width: '0.5em',
                height: '0.5em',
                transform: 'translateY(-50%)',
              }}
            />
          </span>
        </span>

        <span
          className="relative flex items-center justify-center rounded-full border border-[var(--line)] bg-[var(--surface)]"
          style={{ width: '4.5em', height: '4.5em' }}
        >
          <span
            className="absolute bg-[color-mix(in_oklab,var(--sea-ink)_18%,transparent)]"
            style={{ width: '2.5em', height: '0.0625em' }}
          />
          <span
            className="absolute bg-[color-mix(in_oklab,var(--sea-ink)_18%,transparent)]"
            style={{ width: '0.0625em', height: '2.5em' }}
          />
          <span
            className="rounded-full bg-[var(--sea-ink)]"
            style={{ width: '1.2em', height: '1.2em' }}
          />
          <span
            className="absolute rounded-full bg-[var(--palm)]"
            style={{ right: '1em', bottom: '0.8em', width: '0.35em', height: '0.35em' }}
          />
        </span>
      </span>
    </span>
  )
}
