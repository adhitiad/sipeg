import { useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/lib/api'
import {
  pph21TerTarifListSchema,
  pph21TerTarifSchema,
  type Pph21TerTarif,
  type Pph21TerTarifCreate,
} from '@/validations/schemas'

export const pph21TerQueryKey = ['pph21-ter'] as const

/** Membaca seluruh tabel tarif efektif PPh 21 TER. */
export async function fetchPph21Ter(): Promise<Pph21TerTarif[]> {
  const payload = await apiRequest<unknown>('/payroll/pph21-ter')
  return pph21TerTarifListSchema.parse(payload)
}

export function usePph21Ter() {
  return useQuery({
    queryKey: ['pph21-ter', 'list'] as const,
    queryFn: fetchPph21Ter,
    staleTime: 15_000,
    retry: false,
  })
}

export function usePph21TerOn(asOf: string | null) {
  return useQuery({
    queryKey: ['pph21-ter', 'asOf', asOf] as const,
    queryFn: async () => {
      const payload = await apiRequest<unknown>(
        `/payroll/pph21-ter?asOf=${encodeURIComponent(asOf!)}`,
      )
      return pph21TerTarifListSchema.parse(payload)
    },
    enabled: asOf !== null,
    staleTime: 60_000,
    retry: false,
  })
}

export async function createPph21Ter(data: Pph21TerTarifCreate): Promise<Pph21TerTarif> {
  const payload = await apiRequest<unknown>('/payroll/pph21-ter', {
    method: 'POST',
    body: data,
  })
  return pph21TerTarifSchema.parse(payload)
}

export async function updatePph21Ter(
  id: string,
  data: Partial<Pph21TerTarifCreate>,
): Promise<Pph21TerTarif> {
  const payload = await apiRequest<unknown>(`/payroll/pph21-ter/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return pph21TerTarifSchema.parse(payload)
}

export async function deletePph21Ter(id: string): Promise<void> {
  await apiRequest<void>(`/payroll/pph21-ter/${id}`, { method: 'DELETE' })
}
