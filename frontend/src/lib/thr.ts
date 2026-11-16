import { apiRequest } from '@/lib/api'
import {
  thrConfigListSchema,
  thrGenerateReportSchema,
  thrKonfigurasiSchema,
  thrPlanSchema,
  type ThrConfig,
  type ThrKonfigurasiUpsert,
  type ThrPlan,
} from '@/validations/schemas'

export async function fetchThrConfigs(): Promise<ThrConfig[]> {
  const payload = await apiRequest<unknown>('/payroll/thr/config')
  return thrConfigListSchema.parse(payload)
}

export async function upsertThrConfig(data: ThrKonfigurasiUpsert): Promise<ThrConfig> {
  const payload = await apiRequest<unknown>('/payroll/thr/config', {
    method: 'POST',
    body: data,
  })
  return thrKonfigurasiSchema.parse(payload)
}

export async function deleteThrConfig(tahun: number): Promise<void> {
  await apiRequest<null>(`/payroll/thr/config/${tahun}`, { method: 'DELETE' })
}

export async function generateThr(
  tahun: number,
  overrides: Array<{ userId: string; nominal: number }> = [],
): Promise<import('@/validations/schemas').ThrGenerateReport> {
  const payload = await apiRequest<unknown>('/payroll/thr/generate', {
    method: 'POST',
    body: { tahun, overrides },
  })
  return thrGenerateReportSchema.parse(payload)
}

/** Rencana THR: kandidat dengan nominal otomatis dan yang dilewati. */
export async function fetchThrPlan(tahun: number): Promise<ThrPlan> {
  const payload = await apiRequest<unknown>(`/payroll/thr/plan?tahun=${tahun}`)
  return thrPlanSchema.parse(payload)
}
