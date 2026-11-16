import { useState } from 'react'
import { ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from 'sonner'
import { useMessages } from '@/i18n/locale-context'
import { useSession } from '@/lib/session'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchPesangonAturan,
  createPesangonAturan,
  deletePesangonAturan,
} from '@/lib/kontrak'

export default function PesangonAturanPage() {
  const t = useMessages()
  const session = useSession()
  const queryClient = useQueryClient()
  const permissions = session.data?.permissions ?? []
  const canManage = permissions.includes('pesangon:manage')

  const [kategoriStatusId, setKategoriStatusId] = useState('')
  const [masaKerjaTahun, setMasaKerjaTahun] = useState<number | ''>(0)
  const [pesangon, setPesangon] = useState<number | ''>(0)
  const [penghargaan, setPenghargaan] = useState<number | ''>(0)

  const query = useQuery({
    queryKey: ['pesangon-aturan'] as const,
    queryFn: () => fetchPesangonAturan(),
    enabled: canManage,
    staleTime: 30_000,
    retry: false,
  })

  const items = query.data?.items ?? []
  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ['pesangon-aturan'] })

  if (session.data && !canManage) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <ShieldAlert
            className="mx-auto mb-3 h-8 w-8 text-[var(--sea-ink-soft)]"
            aria-hidden="true"
          />
          <h1 className="display-title mb-2 text-2xl font-bold text-[var(--sea-ink)]">
            {t('state.noAccess.title')}
          </h1>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('state.noAccess.description')}
          </p>
        </section>
      </div>
    )
  }

  const handleCreate = async () => {
    try {
      await createPesangonAturan({
        kategoriStatusId: kategoriStatusId.trim(),
        masaKerjaTahun: Number(masaKerjaTahun),
        pesangonBulan: Number(pesangon),
        penghargaanBulan: Number(penghargaan),
        gantiKerugianBulan: 0,
        sumberRegulasi: 'PP 35/2021',
      })
      toast.success(t('pesangon.created'))
      setKategoriStatusId('')
      setMasaKerjaTahun(0)
      setPesangon(0)
      setPenghargaan(0)
      invalidate()
    } catch {
      toast.error(t('pesangon.createError'))
    }
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('pesangon.kicker')}</p>
      <h1 className="display-title mb-2 text-3xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('pesangon.title')}
      </h1>
      <p className="m-0 mb-6 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)]">
        {t('pesangon.lead')}
      </p>

      <div className="island-shell flex flex-col gap-4 rounded-2xl p-4 sm:p-6">
        <div className="grid gap-3 md:grid-cols-5">
          <div className="md:col-span-2">
            <Label htmlFor="kategoriStatusId">
              {t('pesangon.fieldKategoriStatusId')}
            </Label>
            <Input
              id="kategoriStatusId"
              value={kategoriStatusId}
              onChange={(e) => setKategoriStatusId(e.target.value)}
              placeholder="UUID kategori status"
              className="h-11"
            />
          </div>
          <div>
            <Label htmlFor="masaKerjaTahun">{t('pesangon.fieldMasaKerjaTahun')}</Label>
            <Input
              id="masaKerjaTahun"
              type="number"
              min={0}
              value={masaKerjaTahun}
              onChange={(e) =>
                setMasaKerjaTahun(e.target.value === '' ? '' : Number(e.target.value))
              }
              className="h-11"
            />
          </div>
          <div>
            <Label htmlFor="pesangon">{t('pesangon.fieldPesangon')}</Label>
            <Input
              id="pesangon"
              type="number"
              min={0}
              step="0.5"
              value={pesangon}
              onChange={(e) =>
                setPesangon(e.target.value === '' ? '' : Number(e.target.value))
              }
              className="h-11"
            />
          </div>
          <div>
            <Label htmlFor="penghargaan">{t('pesangon.fieldPenghargaan')}</Label>
            <Input
              id="penghargaan"
              type="number"
              min={0}
              step="0.5"
              value={penghargaan}
              onChange={(e) =>
                setPenghargaan(e.target.value === '' ? '' : Number(e.target.value))
              }
              className="h-11"
            />
          </div>
          <div className="flex items-end">
            <Button
              type="button"
              className="w-full"
              disabled={
                !kategoriStatusId ||
                masaKerjaTahun === '' ||
                pesangon === '' ||
                penghargaan === ''
              }
              onClick={handleCreate}
            >
              {t('common.save')}
            </Button>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] border-collapse text-sm">
            <thead>
              <tr className="border-b border-[var(--line)] text-left text-xs uppercase tracking-wide text-[var(--kicker)]">
                <th className="py-2 pr-3">{t('pesangon.colKategori')}</th>
                <th className="py-2 pr-3">{t('pesangon.colMasaKerja')}</th>
                <th className="py-2 pr-3">{t('pesangon.colPesangon')}</th>
                <th className="py-2 pr-3">{t('pesangon.colPenghargaan')}</th>
                <th className="py-2 pr-3 text-right">{t('pesangon.colAksi')}</th>
              </tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-6 text-center text-[var(--sea-ink-soft)]">
                    {t('pesangon.empty')}
                  </td>
                </tr>
              ) : (
                items.map((row) => (
                  <tr key={row.id} className="border-b border-[var(--line)]">
                    <td className="py-2 pr-3 font-mono text-xs">
                      {row.kategoriStatusId}
                    </td>
                    <td className="py-2 pr-3 tabular-nums">{row.masaKerjaTahun}</td>
                    <td className="py-2 pr-3 tabular-nums">{row.pesangonBulan}</td>
                    <td className="py-2 pr-3 tabular-nums">{row.penghargaanBulan}</td>
                    <td className="py-2 pr-3 text-right">
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        onClick={() =>
                          deletePesangonAturan(row.id)
                            .then(invalidate)
                            .catch(() => toast.error(t('pesangon.deleteError')))
                        }
                      >
                        {t('common.delete')}
                      </Button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
