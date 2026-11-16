import { apiRequest } from '@/lib/api'
import {
  publicSlipVerificationSchema,
  type PublicSlipVerification,
} from '@/validations/schemas'

/**
 * Membaca halaman verifikasi publik slip gaji lewat token (dari QR slip).
 * Token bersifat 128-bit acak dan tidak memuat identitas; jawaban hanya
 * berisi nama, periode, dan status — tanpa nominal.
 */
export async function fetchPublicSlipVerification(
  token: string,
): Promise<PublicSlipVerification> {
  const payload = await apiRequest<unknown>(`/verify/slip/${encodeURIComponent(token)}`)
  return publicSlipVerificationSchema.parse(payload)
}
