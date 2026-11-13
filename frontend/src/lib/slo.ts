import { useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/lib/api'
import { sloMetricsSchema, type SloMetrics } from '@/validations/schemas'

/** Rentang pembacaan metrik yang ditawarkan pada halaman admin (jam). */
export const SLO_LOOKBACKS = [24, 168] as const
export type SloLookbackHours = (typeof SLO_LOOKBACKS)[number]

export const sloMetricsQueryKey = (hours: SloLookbackHours) =>
  ['slo-metrics', hours] as const

export async function fetchSloMetrics(hours: SloLookbackHours): Promise<SloMetrics> {
  const payload = await apiRequest<unknown>(`/admin/slo/metrics?hours=${hours}`)
  return sloMetricsSchema.parse(payload)
}

/**
 * Membaca ringkasan metrik SLO untuk halaman admin. Merefresh otomatis
 * setiap 30 detik; kegagalan tidak dicoba ulang agar halaman tidak
 * memukul backend yang sedang bermasalah.
 */
export function useSloMetrics(hours: SloLookbackHours) {
  return useQuery({
    queryKey: sloMetricsQueryKey(hours),
    queryFn: () => fetchSloMetrics(hours),
    staleTime: 30_000,
    retry: false,
  })
}
