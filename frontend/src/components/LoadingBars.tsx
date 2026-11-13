type LoadingBarsSize = 'sm' | 'md' | 'lg'

type LoadingBarsProps = {
  /** Ukuran petak indikator. */
  size?: LoadingBarsSize
  /** Label yang dibacakan pembaca layar. */
  label?: string
  /** Setel true bila teks di dekat indikator sudah menjelaskan keadaan. */
  decorative?: boolean
  className?: string
}

const SIZE_MAP: Record<
  LoadingBarsSize,
  { box: string; bar: string; gap: string; inset: string; radius: string }
> = {
  sm: {
    box: 'h-8 w-8',
    bar: 'h-3 w-1',
    gap: 'gap-[3px]',
    inset: 'inset-[2px]',
    radius: 'rounded-[10px]',
  },
  md: {
    box: 'h-12 w-12',
    bar: 'h-5 w-1.5',
    gap: 'gap-1',
    inset: 'inset-[3px]',
    radius: 'rounded-[var(--radius)]',
  },
  lg: {
    box: 'h-16 w-16',
    bar: 'h-7 w-2',
    gap: 'gap-1',
    inset: 'inset-[4px]',
    radius: 'rounded-[var(--radius)]',
  },
}

/**
 * Indikator memuat berbentuk petak: tepi gradien lagoon-ke-palem berputar
 * sebagai penanda proses, batang di dalam memantul bergantian. Warna diambil
 * dari token produk sehingga tetap terbaca pada tema terang dan gelap.
 *
 * Animasi dimatikan saat pengguna meminta `prefers-reduced-motion`; bentuk
 * petak dan batang tetap terlihat sebagai tanda bahwa data sedang dimuat.
 */
export default function LoadingBars({
  size = 'md',
  label = 'Memuat',
  decorative = false,
  className = '',
}: LoadingBarsProps) {
  const dimensions = SIZE_MAP[size]

  return (
    <span
      role={decorative ? undefined : 'status'}
      aria-live={decorative ? undefined : 'polite'}
      aria-busy={decorative ? undefined : 'true'}
      aria-label={decorative ? undefined : label}
      aria-hidden={decorative ? 'true' : undefined}
      className={`inline-flex ${className}`}
    >
      <span
        aria-hidden="true"
        className={`relative inline-flex items-center justify-center ${dimensions.box}`}
      >
        <span
          className={`absolute inset-0 ${dimensions.radius} bg-[conic-gradient(from_0deg,var(--lagoon),var(--palm),var(--lagoon))] motion-safe:animate-[spin_2.4s_linear_infinite]`}
        />
        <span
          className={`absolute ${dimensions.inset} ${dimensions.radius} bg-[var(--surface-strong)]`}
        />
        <span className={`relative flex items-end ${dimensions.gap}`}>
          <span
            className={`${dimensions.bar} rounded-full bg-[var(--lagoon)] motion-safe:animate-[bounce_1s_ease-in-out_infinite]`}
            style={{ animationDelay: '0s' }}
          />
          <span
            className={`${dimensions.bar} rounded-full bg-[var(--palm)] motion-safe:animate-[bounce_1s_ease-in-out_infinite]`}
            style={{ animationDelay: '0.12s' }}
          />
          <span
            className={`${dimensions.bar} rounded-full bg-[var(--lagoon-deep)] motion-safe:animate-[bounce_1s_ease-in-out_infinite]`}
            style={{ animationDelay: '0.24s' }}
          />
        </span>
      </span>
    </span>
  )
}
