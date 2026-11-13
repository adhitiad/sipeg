import { apiRequest } from '@/lib/api'
import {
  employeeSummarySchema,
  type EmployeeSummaryResponse,
  type Organization,
} from '@/validations/schemas'

export type EmployeeSummary = EmployeeSummaryResponse

/**
 * Membaca ringkasan pegawai dari GET /api/v1/employees/summary.
 * Backend menegakkan cakupan principal (employee:read:any/:homebase/:self);
 * frontend tidak menambah pembatasan lain.
 */
export async function fetchEmployeeSummary(): Promise<EmployeeSummary> {
  const payload = await apiRequest<unknown>('/employees/summary')
  return employeeSummarySchema.parse(payload)
}

/** Memetakan id unit ke nama unit untuk ringkasan. */
export function resolveUnitNames(
  summary: EmployeeSummary,
  units: Organization[],
): Array<{ unitId: string | null; unitName: string; count: number }> {
  const nameMap = new Map<string, string>()
  for (const unit of units) {
    if (unit.name) {
      nameMap.set(unit.id, unit.name)
    }
  }
  return summary.byUnit.map((entry) => ({
    unitId: entry.unitId,
    unitName: entry.unitId ? (nameMap.get(entry.unitId) ?? entry.unitId) : 'Tanpa unit',
    count: entry.count,
  }))
}
