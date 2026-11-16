import { apiRequest } from '@/lib/api'
import {
  kompensasiPreviewSchema,
  pemberhentianListSchema,
  pemberhentianSchema,
  type KompensasiPreview,
  type Pemberhentian,
  type PemberhentianPropose,
} from '@/validations/schemas'

/** Daftar pengajuan/keputusan pemberhentian untuk halaman admin. */
export async function fetchPemberhentian(params: {
  userId?: string
  status?: string
}): Promise<{ items: Pemberhentian[]; total: number }> {
  const search = new URLSearchParams()
  if (params.userId) search.set('userId', params.userId)
  if (params.status) search.set('status', params.status)
  const query = search.toString()
  const payload = await apiRequest<unknown>(`/pemberhentian${query ? `?${query}` : ''}`)
  return pemberhentianListSchema.parse(payload)
}

/** Ajukan pemberhentian pegawai. */
export async function proposePemberhentian(
  data: PemberhentianPropose,
): Promise<Pemberhentian> {
  const payload = await apiRequest<unknown>('/pemberhentian', {
    method: 'POST',
    body: data,
  })
  return pemberhentianSchema.parse(payload)
}

/** Pratinjau kompensasi pemberhentian (sebelum approve); tidak menulis apa pun. */
export async function fetchKompensasiPreview(id: string): Promise<KompensasiPreview> {
  const payload = await apiRequest<unknown>(`/pemberhentian/${id}/kompensasi-preview`)
  return kompensasiPreviewSchema.parse(payload)
}

/** Setujui pengajuan pemberhentian (admin). */
export async function approvePemberhentian(
  id: string,
  note?: string,
): Promise<Pemberhentian> {
  const payload = await apiRequest<unknown>(`/pemberhentian/${id}/approve`, {
    method: 'POST',
    body: { note: note ?? null },
  })
  return pemberhentianSchema.parse(payload)
}

/** Tolak pengajuan pemberhentian (admin). */
export async function rejectPemberhentian(
  id: string,
  note?: string,
): Promise<Pemberhentian> {
  const payload = await apiRequest<unknown>(`/pemberhentian/${id}/reject`, {
    method: 'POST',
    body: { note: note ?? null },
  })
  return pemberhentianSchema.parse(payload)
}
