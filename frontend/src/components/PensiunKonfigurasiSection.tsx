import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { toast } from 'sonner'
import { useMessages } from '@/i18n/locale-context'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError } from '@/lib/api'
import {
  deletePensiunKonfigurasi,
  fetchPensiunKonfigurasi,
  updatePensiunKonfigurasi,
  upsertPensiunKonfigurasi,
  type PensiunKonfigurasiInput,
} from '@/lib/pensiun'
import { useKategoriStatus } from '@/lib/payroll-scale'
import type { PensiunKonfigurasi } from '@/validations/schemas'

export default function PensiunKonfigurasiSection({
  canWrite = true,
}: { canWrite?: boolean } = {}) {
  const t = useMessages()
  const queryClient = useQueryClient()
  const { data: kategoriStatuses = [] } = useKategoriStatus()

  const [kategoriStatusId, setKategoriStatusId] = useState('')
  const [bupUmur, setBupUmur] = useState<number | ''>(65)
  const [dplkWajib, setDplkWajib] = useState(true)
  const [iuranPegawai, setIuranPegawai] = useState<number | ''>(5)
  const [iuranPemberiKerja, setIuranPemberiKerja] = useState<number | ''>(10)
  const [penyedia, setPenyedia] = useState('')
  const [berlakuMulai, setBerlakuMulai] = useState('')
  const [editId, setEditId] = useState<string | null>(null)

  const query = useQuery({
    queryKey: ['pensiun', 'konfigurasi'] as const,
    queryFn: () => fetchPensiunKonfigurasi(),
    staleTime: 30_000,
    retry: false,
  })
  const items = query.data ?? []

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ['pensiun', 'konfigurasi'] })

  const kategoriNama = (id: string) =>
    kategoriStatuses.find((k) => k.id === id)?.nama ?? id

  const errorMessage = (err: unknown) =>
    err instanceof ApiError && err.message ? err.message : t('pensiun.failed')

  const formValid =
    kategoriStatusId !== '' &&
    typeof bupUmur === 'number' &&
    bupUmur >= 45 &&
    bupUmur <= 80 &&
    typeof iuranPegawai === 'number' &&
    iuranPegawai >= 0 &&
    iuranPegawai <= 100 &&
    typeof iuranPemberiKerja === 'number' &&
    iuranPemberiKerja >= 0 &&
    iuranPemberiKerja <= 100 &&
    berlakuMulai !== ''

  const buildInput = (): PensiunKonfigurasiInput => ({
    kategoriStatusId,
    bupUmur: Number(bupUmur),
    dplkWajib,
    iuranPegawaiPct: Number(iuranPegawai),
    iuranPemberiKerjaPct: Number(iuranPemberiKerja),
    penyediaDplk: penyedia.trim() || undefined,
    berlakuMulai,
    aktif: true,
  })

  const resetForm = () => {
    setKategoriStatusId('')
    setBupUmur(65)
    setDplkWajib(true)
    setIuranPegawai(5)
    setIuranPemberiKerja(10)
    setPenyedia('')
    setBerlakuMulai('')
    setEditId(null)
  }

  const handleEdit = (item: PensiunKonfigurasi) => {
    setEditId(item.id)
    setKategoriStatusId(item.kategoriStatusId)
    setBupUmur(item.bupUmur)
    setDplkWajib(item.dplkWajib)
    setIuranPegawai(item.iuranPegawaiPct)
    setIuranPemberiKerja(item.iuranPemberiKerjaPct)
    setPenyedia(item.penyediaDplk ?? '')
    setBerlakuMulai(item.berlakuMulai)
  }

  const handleSave = async () => {
    if (!formValid) {
      toast.error(t('pensiun.invalid'))
      return
    }
    try {
      if (editId) {
        await updatePensiunKonfigurasi(editId, buildInput())
      } else {
        await upsertPensiunKonfigurasi(buildInput())
      }
      toast.success(t('pensiun.saved'))
      resetForm()
      invalidate()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deletePensiunKonfigurasi(id)
      toast.success(t('pensiun.deleted'))
      invalidate()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  return (
    <section className="island-shell rounded-2xl p-5">
      <h2 className="mb-3 text-lg font-semibold text-[var(--sea-ink)]">
        {t('pensiun.konfigurasi')}
      </h2>
      {canWrite && (
        <div className="mb-4 grid gap-2 lg:grid-cols-3">
          <div>
            <Label htmlFor="pensiun-kat">{t('pensiun.kategori')}</Label>
            <Select value={kategoriStatusId} onValueChange={setKategoriStatusId}>
              <SelectTrigger id="pensiun-kat" className="mt-1 w-full">
                <SelectValue placeholder={t('pensiun.kategori')} />
              </SelectTrigger>
              <SelectContent>
                {kategoriStatuses.map((k) => (
                  <SelectItem key={k.id} value={k.id}>
                    {k.nama}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label htmlFor="pensiun-bup">{t('pensiun.bup')}</Label>
            <Input
              id="pensiun-bup"
              type="number"
              min={45}
              max={80}
              value={bupUmur}
              onChange={(e) =>
                setBupUmur(e.target.value === '' ? '' : Number(e.target.value))
              }
            />
          </div>
          <div>
            <Label htmlFor="pensiun-berlaku">{t('pensiun.berlakuMulai')}</Label>
            <Input
              id="pensiun-berlaku"
              type="date"
              value={berlakuMulai}
              onChange={(e) => setBerlakuMulai(e.target.value)}
            />
          </div>
          <div>
            <Label htmlFor="pensiun-iuran-pg">{t('pensiun.iuranPegawai')}</Label>
            <Input
              id="pensiun-iuran-pg"
              type="number"
              min={0}
              max={100}
              step={0.01}
              value={iuranPegawai}
              onChange={(e) =>
                setIuranPegawai(e.target.value === '' ? '' : Number(e.target.value))
              }
            />
          </div>
          <div>
            <Label htmlFor="pensiun-iuran-pk">{t('pensiun.iuranPemberiKerja')}</Label>
            <Input
              id="pensiun-iuran-pk"
              type="number"
              min={0}
              max={100}
              step={0.01}
              value={iuranPemberiKerja}
              onChange={(e) =>
                setIuranPemberiKerja(e.target.value === '' ? '' : Number(e.target.value))
              }
            />
          </div>
          <div>
            <Label htmlFor="pensiun-penyedia">{t('pensiun.penyedia')}</Label>
            <Input
              id="pensiun-penyedia"
              value={penyedia}
              onChange={(e) => setPenyedia(e.target.value)}
              placeholder="DPLK Bank Mandiri"
            />
          </div>
          <div className="flex items-center gap-2 self-end pb-1">
            <Checkbox
              id="pensiun-wajib"
              checked={dplkWajib}
              onCheckedChange={(v) => setDplkWajib(v === true)}
            />
            <Label htmlFor="pensiun-wajib">{t('pensiun.dplkWajib')}</Label>
          </div>
          <div className="flex items-end lg:col-span-2">
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
                <span className="font-medium">{kategoriNama(item.kategoriStatusId)}</span>
                <Badge variant="outline">BUP {item.bupUmur}</Badge>
                <span className="text-[var(--sea-ink-soft)]">
                  {item.iuranPegawaiPct}% / {item.iuranPemberiKerjaPct}% ·{' '}
                  {item.penyediaDplk ?? '-'} · {item.berlakuMulai}
                </span>
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
                    onClick={() => void handleDelete(item.id)}
                    aria-label={t('pensiun.delete')}
                  >
                    <Trash2 className="h-4 w-4" aria-hidden="true" />
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
