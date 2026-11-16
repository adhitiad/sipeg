import { apiRequest } from '@/lib/api'
import {
  kontrakListSchema,
  kontrakSchema,
  pesangonAturanListSchema,
  pesangonAturanSchema,
  type Kontrak,
  type Perpanjangan,
} from '@/validations/schemas'

/** Daftar kontrak kerja. */
export async function fetchKontrak(params: {
  userId?: string
  jenis?: string
  mulaiSesudah?: string
  selesaiBefore?: string
}): Promise<{ items: Kontrak[]; total: number }> {
  const search = new URLSearchParams()
  if (params.userId) search.set('userId', params.userId)
  if (params.jenis) search.set('jenis', params.jenis)
  if (params.selesaiBefore) search.set('selesaiBefore', params.selesaiBefore)
  const query = search.toString()
  const payload = await apiRequest<unknown>(`/kontrak${query ? `?${query}` : ''}`)
  return kontrakListSchema.parse(payload)
}

/** Perpanjangan perjanjian kerja tahunan pasca-BUP (admin). */
export async function extendPerpanjangan(data: Perpanjangan): Promise<Kontrak> {
  const payload = await apiRequest<unknown>('/perpanjangan', {
    method: 'POST',
    body: data,
  })
  return kontrakSchema.parse(payload)
}

/** Daftar aturan pesangon (admin). */
export async function fetchPesangonAturan(kategoriStatusId?: string) {
  const query = kategoriStatusId ? `?kategoriStatusId=${kategoriStatusId}` : ''
  const payload = await apiRequest<unknown>(`/pesangon-aturan${query}`)
  return pesangonAturanListSchema.parse(payload)
}

export async function createPesangonAturan(data: {
  kategoriStatusId: string
  masaKerjaTahun: number
  pesangonBulan: number
  penghargaanBulan: number
  gantiKerugianBulan: number
  sumberRegulasi?: string | null
}) {
  const payload = await apiRequest<unknown>('/pesangon-aturan', {
    method: 'POST',
    body: data,
  })
  return pesangonAturanSchema.parse(payload)
}

export async function updatePesangonAturan(
  id: string,
  data: {
    pesangonBulan: number
    penghargaanBulan: number
    gantiKerugianBulan: number
    sumberRegulasi: string
    kategoriStatusId: string
    masaKerjaTahun: number
  },
) {
  const payload = await apiRequest<unknown>(`/pesangon-aturan/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return pesangonAturanSchema.parse(payload)
}

export async function deletePesangonAturan(id: string): Promise<void> {
  await apiRequest<null>(`/pesangon-aturan/${id}`, { method: 'DELETE' })
}
