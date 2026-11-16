import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { toast } from 'sonner'
import { useMessages } from '@/i18n/locale-context'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError } from '@/lib/api'
import {
  deactivatePensiunKepesertaan,
  fetchPensiunKepesertaan,
  updatePensiunKepesertaan,
  upsertPensiunKepesertaan,
  type PensiunKepesertaanInput,
} from '@/lib/pensiun'
import type { PensiunKepesertaan } from '@/validations/schemas'

export default function PensiunKepesertaanSection({
  canWrite = true,
}: { canWrite?: boolean } = {}) {
  const t = useMessages()
  const queryClient = useQueryClient()

  const [userId, setUserId] = useState('')
  const [penyedia, setPenyedia] = useState('')
  const [nomor, setNomor] = useState('')
  const [mulai, setMulai] = useState('')
  const [tanggalMulaiPensiun, setTanggalMulaiPensiun] = useState('')
  const [aktif, setAktif] = useState(true)
  const [editId, setEditId] = useState<string | null>(null)

  const query = useQuery({
    queryKey: ['pensiun', 'kepesertaan'] as const,
    queryFn: fetchPensiunKepesertaan,
    staleTime: 30_000,
    retry: false,
  })
  const items = query.data ?? []

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ['pensiun', 'kepesertaan'] })

  const errorMessage = (err: unknown) =>
    err instanceof ApiError && err.message ? err.message : t('pensiun.failed')

  const formValid = userId.trim() !== '' && penyedia.trim() !== '' && mulai !== ''

  const buildInput = (): PensiunKepesertaanInput => ({
    userId: userId.trim(),
    penyediaDplk: penyedia.trim(),
    nomorKepesertaan: nomor.trim() || undefined,
    mulai,
    tanggalMulaiPensiun: tanggalMulaiPensiun || undefined,
    aktif,
  })

  const resetForm = () => {
    setUserId('')
    setPenyedia('')
    setNomor('')
    setMulai('')
    setTanggalMulaiPensiun('')
    setAktif(true)
    setEditId(null)
  }

  const handleEdit = (item: PensiunKepesertaan) => {
    setEditId(item.id)
    setUserId(item.userId)
    setPenyedia(item.penyediaDplk)
    setNomor(item.nomorKepesertaan ?? '')
    setMulai(item.mulai)
    setTanggalMulaiPensiun(item.tanggalMulaiPensiun ?? '')
    setAktif(item.aktif)
  }

  const handleSave = async () => {
    if (!formValid) {
      toast.error(t('pensiun.invalid'))
      return
    }
    try {
      if (editId) {
        await updatePensiunKepesertaan(editId, buildInput())
      } else {
        await upsertPensiunKepesertaan(buildInput())
      }
      toast.success(t('pensiun.saved'))
      resetForm()
      invalidate()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  const handleDeactivate = async (id: string) => {
    try {
      await deactivatePensiunKepesertaan(id)
      toast.success(t('pensiun.deactivated'))
      invalidate()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  return (
    <section className="island-shell rounded-2xl p-5">
      <h2 className="mb-3 text-lg font-semibold text-[var(--sea-ink)]">
        {t('pensiun.kepesertaan')}
      </h2>
      {canWrite && (
        <div className="mb-4 grid gap-2 lg:grid-cols-3">
          <div>
            <Label htmlFor="pensiun-user">{t('pensiun.userId')}</Label>
            <Input
              id="pensiun-user"
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              placeholder="UUID"
            />
          </div>
          <div>
            <Label htmlFor="pensiun-kep-penyedia">{t('pensiun.penyedia')}</Label>
            <Input
              id="pensiun-kep-penyedia"
              value={penyedia}
              onChange={(e) => setPenyedia(e.target.value)}
              placeholder="DPLK Bank Mandiri"
            />
          </div>
          <div>
            <Label htmlFor="pensiun-nomor">{t('pensiun.nomor')}</Label>
            <Input
              id="pensiun-nomor"
              value={nomor}
              onChange={(e) => setNomor(e.target.value)}
            />
          </div>
          <div>
            <Label htmlFor="pensiun-mulai">{t('pensiun.mulai')}</Label>
            <Input
              id="pensiun-mulai"
              type="date"
              value={mulai}
              onChange={(e) => setMulai(e.target.value)}
            />
          </div>
          <div>
            <Label htmlFor="pensiun-tgl-pensiun">
              {t('pensiun.tanggalMulaiPensiun')}
            </Label>
            <Input
              id="pensiun-tgl-pensiun"
              type="date"
              value={tanggalMulaiPensiun}
              onChange={(e) => setTanggalMulaiPensiun(e.target.value)}
            />
            <p className="mt-1 text-xs text-[var(--sea-ink-soft)]">
              {t('pensiun.tanggalHint')}
            </p>
          </div>
          <div className="flex items-center gap-2 self-end pb-1">
            <Checkbox
              id="pensiun-aktif"
              checked={aktif}
              onCheckedChange={(v) => setAktif(v === true)}
            />
            <Label htmlFor="pensiun-aktif">{t('pensiun.aktif')}</Label>
          </div>
          <div className="flex items-end lg:col-span-3">
            <Button type="button" onClick={() => void handleSave()} disabled={!formValid}>
              {editId ? t('pensiun.save') : t('pensiun.add')}
            </Button>
          </div>
        </div>
      )}

      <div className="space-y-2">
        {items.length === 0 ? (
          <p className="text-sm text-[var(--sea-ink-soft)]">{t('pensiun.empty')}</p>
        ) : (
          items.map((item) => (
            <div
              key={item.id}
              className="flex items-center justify-between gap-2 rounded-lg border border-[var(--line)] p-2 text-sm"
            >
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{item.nama ?? item.userId}</span>
                <span className="text-[var(--sea-ink-soft)]">{item.penyediaDplk}</span>
                <span className="text-[var(--sea-ink-soft)]">
                  {item.nomorKepesertaan ?? '-'}
                </span>
                <span className="text-[var(--sea-ink-soft)]">{item.mulai}</span>
                {item.tanggalMulaiPensiun ? (
                  <Badge variant="outline">
                    {t('pensiun.tanggalMulaiPensiun')} {item.tanggalMulaiPensiun}
                  </Badge>
                ) : null}
                {item.aktif ? (
                  <Badge variant="secondary">{t('pensiun.aktif')}</Badge>
                ) : null}
              </div>
              {canWrite && (
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => handleEdit(item)}
                  >
                    {t('pensiun.edit')}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => void handleDeactivate(item.id)}
                  >
                    {t('pensiun.deactivate')}
                  </Button>
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </section>
  )
}
