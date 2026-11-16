import { apiRequest } from '@/lib/api'
import {
  pensiunKepesertaanListSchema,
  pensiunKepesertaanSchema,
  pensiunKonfigurasiListSchema,
  pensiunKonfigurasiSchema,
  type PensiunKepesertaan,
  type PensiunKonfigurasi,
} from '@/validations/schemas'

export async function fetchPensiunKonfigurasi(
  kategoriStatusId?: string,
): Promise<PensiunKonfigurasi[]> {
  const query = kategoriStatusId
    ? `?kategoriStatusId=${encodeURIComponent(kategoriStatusId)}`
    : ''
  const payload = await apiRequest<unknown>(`/pensiun/konfigurasi${query}`)
  return pensiunKonfigurasiListSchema.parse(payload).items
}

export interface PensiunKonfigurasiInput {
  kategoriStatusId: string
  bupUmur: number
  dplkWajib: boolean
  iuranPegawaiPct: number
  iuranPemberiKerjaPct: number
  penyediaDplk?: string
  berlakuMulai: string
  aktif: boolean
}

export async function upsertPensiunKonfigurasi(
  data: PensiunKonfigurasiInput,
): Promise<PensiunKonfigurasi> {
  const payload = await apiRequest<unknown>('/pensiun/konfigurasi', {
    method: 'POST',
    body: data,
  })
  return pensiunKonfigurasiSchema.parse(payload)
}

export async function updatePensiunKonfigurasi(
  id: string,
  data: PensiunKonfigurasiInput,
): Promise<PensiunKonfigurasi> {
  const payload = await apiRequest<unknown>(`/pensiun/konfigurasi/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return pensiunKonfigurasiSchema.parse(payload)
}

export async function deletePensiunKonfigurasi(id: string): Promise<void> {
  await apiRequest<null>(`/pensiun/konfigurasi/${id}`, { method: 'DELETE' })
}

export async function fetchPensiunKepesertaan(): Promise<PensiunKepesertaan[]> {
  const payload = await apiRequest<unknown>('/pensiun/kepesertaan')
  return pensiunKepesertaanListSchema.parse(payload).items
}

export interface PensiunKepesertaanInput {
  userId: string
  penyediaDplk: string
  nomorKepesertaan?: string
  mulai: string
  tanggalMulaiPensiun?: string
  aktif: boolean
}

export async function upsertPensiunKepesertaan(
  data: PensiunKepesertaanInput,
): Promise<PensiunKepesertaan> {
  const payload = await apiRequest<unknown>('/pensiun/kepesertaan', {
    method: 'POST',
    body: data,
  })
  return pensiunKepesertaanSchema.parse(payload)
}

export async function updatePensiunKepesertaan(
  id: string,
  data: PensiunKepesertaanInput,
): Promise<PensiunKepesertaan> {
  const payload = await apiRequest<unknown>(`/pensiun/kepesertaan/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return pensiunKepesertaanSchema.parse(payload)
}

export async function deactivatePensiunKepesertaan(id: string): Promise<void> {
  await apiRequest<null>(`/pensiun/kepesertaan/${id}`, { method: 'DELETE' })
}
