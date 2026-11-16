import { apiRequest } from '@/lib/api'
import { skListSchema, skSchema, type Sk, type SkList } from '@/validations/schemas'

export interface SkListParams {
  search?: string
  jenis?: string
  penerbit?: string
  page?: number
  pageSize?: number
}

/** Membaca daftar surat keputusan sesuai filter dan cakupan principal. */
export async function fetchSkList(params: SkListParams = {}): Promise<SkList> {
  const query = new URLSearchParams()
  if (params.search) query.set('search', params.search)
  if (params.jenis) query.set('jenis', params.jenis)
  if (params.penerbit) query.set('penerbit', params.penerbit)
  if (params.page) query.set('page', String(params.page))
  if (params.pageSize) query.set('pageSize', String(params.pageSize))
  const suffix = query.toString()
  const payload = await apiRequest<unknown>(
    `/surat-keputusan${suffix ? `?${suffix}` : ''}`,
  )
  return skListSchema.parse(payload)
}

/** Membaca detail satu surat keputusan. */
export async function fetchSk(id: string): Promise<Sk> {
  const payload = await apiRequest<unknown>(`/surat-keputusan/${id}`)
  return skSchema.parse(payload)
}
