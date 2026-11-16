import { ClipboardList, FileSignature } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useMessages } from '@/i18n/locale-context'
import { useSession } from '@/lib/session'
import { fetchKontrak } from '@/lib/kontrak'
import { fetchPemberhentian } from '@/lib/pemberhentian'

/**
 * Riwayat pemberhentian dan kontrak seorang pegawai — ditampilkan pada
 * halaman detail pegawai. Hanya dirender bila principal memegang permission
 * membaca data tersebut (karyawan lain/atasan/operator/admin).
 */
export default function EmployeeRiwayatCards({ userId }: { userId: string }) {
  const t = useMessages()
  const session = useSession()
  const permissions = session.data?.permissions ?? []
  const canReadPemberhentian =
    permissions.includes('pemberhentian:read:any') ||
    permissions.includes('pemberhentian:read:homebase')
  const canReadKontrak =
    permissions.includes('kontrak:read:any') ||
    permissions.includes('kontrak:read:homebase') ||
    permissions.includes('kontrak:read:self')

  const pemberhentian = useQuery({
    queryKey: ['pemberhentian', 'riwayat', userId] as const,
    queryFn: () => fetchPemberhentian({ userId }),
    enabled: Boolean(userId) && canReadPemberhentian,
    staleTime: 30_000,
    retry: false,
  })
  const kontrak = useQuery({
    queryKey: ['kontrak', 'riwayat', userId] as const,
    queryFn: () => fetchKontrak({ userId }),
    enabled: Boolean(userId) && canReadKontrak,
    staleTime: 30_000,
    retry: false,
  })

  if (!canReadPemberhentian && !canReadKontrak) return null

  return (
    <div className="mt-6 grid gap-6 lg:grid-cols-2">
      {canReadPemberhentian ? (
        <Card className="island-shadow">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ClipboardList className="h-5 w-5" aria-hidden="true" />
              {t('employee.riwayatPemberhentianTitle')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {pemberhentian.isPending ? (
              <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
                {t('common.loading')}
              </p>
            ) : pemberhentian.data?.items.length ? (
              <div className="overflow-x-auto">
                <table className="w-full min-w-[420px] border-collapse text-sm">
                  <thead>
                    <tr className="border-b border-[var(--line)] text-left text-xs uppercase tracking-wide text-[var(--kicker)]">
                      <th className="py-2 pr-3">{t('pemberhentian.colTanggal')}</th>
                      <th className="py-2 pr-3">{t('pemberhentian.colAlasan')}</th>
                      <th className="py-2 pr-3">{t('pemberhentian.colStatus')}</th>
                      <th className="py-2 pr-3 text-right">
                        {t('pemberhentian.colKompensasi')}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {pemberhentian.data.items.map((item) => (
                      <tr key={item.id} className="border-b border-[var(--line)]">
                        <td className="py-2 pr-3 tabular-nums">{item.tanggal}</td>
                        <td className="py-2 pr-3">{item.alasan}</td>
                        <td className="py-2 pr-3">
                          <Badge
                            variant={
                              item.status === 'DISETUJUI' ? 'default' : 'secondary'
                            }
                          >
                            {item.status}
                          </Badge>
                        </td>
                        <td className="py-2 pr-3 text-right tabular-nums">
                          {item.kompensasiBulan > 0 ? String(item.kompensasiBulan) : '—'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
                {t('employee.riwayatEmpty')}
              </p>
            )}
          </CardContent>
        </Card>
      ) : null}
      {canReadKontrak ? (
        <Card className="island-shadow">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileSignature className="h-5 w-5" aria-hidden="true" />
              {t('employee.riwayatKontrakTitle')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {kontrak.isPending ? (
              <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
                {t('common.loading')}
              </p>
            ) : kontrak.data?.items.length ? (
              <div className="overflow-x-auto">
                <table className="w-full min-w-[420px] border-collapse text-sm">
                  <thead>
                    <tr className="border-b border-[var(--line)] text-left text-xs uppercase tracking-wide text-[var(--kicker)]">
                      <th className="py-2 pr-3">{t('kontrak.colJenis')}</th>
                      <th className="py-2 pr-3">{t('kontrak.colMulai')}</th>
                      <th className="py-2 pr-3">{t('kontrak.colSelesai')}</th>
                      <th className="py-2 pr-3">{t('kontrak.colStatus')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {kontrak.data.items.map((item) => (
                      <tr key={item.id} className="border-b border-[var(--line)]">
                        <td className="py-2 pr-3 text-xs">{item.jenis}</td>
                        <td className="py-2 pr-3 tabular-nums">{item.mulai}</td>
                        <td className="py-2 pr-3 tabular-nums">{item.selesai ?? '—'}</td>
                        <td className="py-2 pr-3">{item.status}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
                {t('employee.riwayatEmpty')}
              </p>
            )}
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}
