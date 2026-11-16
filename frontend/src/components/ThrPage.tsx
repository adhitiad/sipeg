import { useState } from 'react'
import { ShieldAlert, Sparkles, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { toast } from 'sonner'
import { useMessages } from '@/i18n/locale-context'
import { useSession } from '@/lib/session'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchThrConfigs,
  fetchThrPlan,
  upsertThrConfig,
  deleteThrConfig,
  generateThr,
} from '@/lib/thr'
import type { ThrGenerateReport, ThrPlan } from '@/validations/schemas'

const BESARAN = ['SATU_GAJI_POKOK', 'GAJI_KOTOR', 'PERSEN', 'PERSEN_TAKE_HOME'] as const
type Besaran = (typeof BESARAN)[number]

export default function ThrPage() {
  const t = useMessages()
  const session = useSession()
  const queryClient = useQueryClient()
  const permissions = session.data?.permissions ?? []
  const canManage = permissions.includes('thr:manage')

  const [tahun, setTahun] = useState<number | ''>(new Date().getFullYear())
  const [besaranTipe, setBesaranTipe] = useState<Besaran>('GAJI_KOTOR')
  const [persen, setPersen] = useState<number | ''>(100)
  const [syarat, setSyarat] = useState<number | ''>(3)
  const [berlakuTanggal, setBerlakuTanggal] = useState('')
  const [generating, setGenerating] = useState(false)
  const [report, setReport] = useState<ThrGenerateReport | null>(null)
  const [plan, setPlan] = useState<ThrPlan | null>(null)
  const [nominals, setNominals] = useState<Record<string, number>>({})
  const [loadingPlan, setLoadingPlan] = useState(false)

  const query = useQuery({
    queryKey: ['thr', 'config'] as const,
    queryFn: fetchThrConfigs,
    enabled: canManage,
    staleTime: 15_000,
    retry: false,
  })

  const items = query.data ?? []
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['thr', 'config'] })

  const handleCreate = async () => {
    try {
      await upsertThrConfig({
        tahun: Number(tahun),
        besaranTipe,
        persen:
          besaranTipe === 'PERSEN' || besaranTipe === 'PERSEN_TAKE_HOME'
            ? Number(persen) || null
            : null,
        syaratMasaKerjaBulan: Number(syarat),
        berlakuTanggal,
      })
      toast.success(t('thr.saved'))
      setBerlakuTanggal('')
      invalidate()
    } catch {
      toast.error(t('thr.failed'))
    }
  }

  const handleLoadPlan = async () => {
    setLoadingPlan(true)
    try {
      const result = await fetchThrPlan(Number(tahun))
      setPlan(result)
      const awal: Record<string, number> = {}
      for (const item of result.items) {
        awal[item.userId] = item.nominalOtomatis
      }
      setNominals(awal)
    } catch {
      toast.error(t('thr.planFailed'))
    } finally {
      setLoadingPlan(false)
    }
  }

  const handleGenerate = async () => {
    setGenerating(true)
    setReport(null)
    try {
      // Hanya nominal yang diubah admin yang dikirim sebagai override
      // (prorata); sisanya memakai nominal otomatis.
      const overrides = (plan?.items ?? [])
        .filter((item) => nominals[item.userId] !== item.nominalOtomatis)
        .map((item) => ({
          userId: item.userId,
          nominal: Number(nominals[item.userId]) || 0,
        }))
        .filter((item) => item.nominal > 0)
      const result = await generateThr(Number(tahun), overrides)
      setReport(result)
      toast.success(t('thr.generated', { count: String(result.generated) }))
      invalidate()
    } catch {
      toast.error(t('thr.generateFailed'))
    } finally {
      setGenerating(false)
    }
  }

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

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('thr.kicker')}</p>
      <h1 className="display-title mb-2 text-3xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('thr.title')}
      </h1>
      <p className="m-0 mb-6 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)]">
        {t('thr.lead')}
      </p>

      <div className="island-shell flex flex-col gap-6 rounded-2xl p-4 sm:p-6">
        <section aria-label={t('thr.formTitle')} className="grid gap-3 md:grid-cols-6">
          <div>
            <Label htmlFor="tahun">{t('thr.fieldTahun')}</Label>
            <Input
              id="tahun"
              type="number"
              min={2026}
              value={tahun}
              onChange={(e) =>
                setTahun(e.target.value === '' ? '' : Number(e.target.value))
              }
              className="h-11"
            />
          </div>
          <div className="md:col-span-2">
            <Label htmlFor="besaran">{t('thr.fieldBesaran')}</Label>
            <select
              id="besaran"
              value={besaranTipe}
              onChange={(e) => setBesaranTipe(e.target.value as Besaran)}
              className="flex h-11 w-full rounded-md border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 text-sm text-[var(--sea-ink)]"
            >
              {BESARAN.map((value) => (
                <option key={value} value={value}>
                  {t(`thr.besaran.${value}` as never)}
                </option>
              ))}
            </select>
          </div>
          {besaranTipe === 'PERSEN' || besaranTipe === 'PERSEN_TAKE_HOME' ? (
            <div>
              <Label htmlFor="persen">{t('thr.fieldPersen')}</Label>
              <Input
                id="persen"
                type="number"
                min={0}
                max={400}
                value={persen}
                onChange={(e) =>
                  setPersen(e.target.value === '' ? '' : Number(e.target.value))
                }
                className="h-11"
              />
            </div>
          ) : null}
          <div>
            <Label htmlFor="syarat">{t('thr.fieldSyarat')}</Label>
            <Input
              id="syarat"
              type="number"
              min={0}
              max={60}
              value={syarat}
              onChange={(e) =>
                setSyarat(e.target.value === '' ? '' : Number(e.target.value))
              }
              className="h-11"
            />
          </div>
          <div>
            <Label htmlFor="berlaku">{t('thr.fieldBerlaku')}</Label>
            <Input
              id="berlaku"
              type="date"
              value={berlakuTanggal}
              onChange={(e) => setBerlakuTanggal(e.target.value)}
              className="h-11"
            />
          </div>
          <div className="flex items-end md:col-span-2">
            <Button
              type="button"
              className="w-full"
              disabled={!tahun || !berlakuTanggal || syarat === ''}
              onClick={handleCreate}
            >
              {t('common.save')}
            </Button>
          </div>
        </section>

        <section
          aria-label={t('thr.generateTitle')}
          className="border-t border-[var(--line)] pt-4"
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="m-0 text-lg font-semibold text-[var(--sea-ink)]">
                {t('thr.generateTitle')}
              </h2>
              <p className="m-0 max-w-xl text-sm text-[var(--sea-ink-soft)]">
                {t('thr.generateLead')}
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                variant="outline"
                disabled={loadingPlan || !tahun}
                onClick={handleLoadPlan}
              >
                {loadingPlan ? t('common.loading') : t('thr.loadPlan')}
              </Button>
              <Button
                type="button"
                disabled={generating || !tahun}
                onClick={handleGenerate}
                className="gap-2"
              >
                <Sparkles className="h-4 w-4" aria-hidden="true" />
                {generating ? t('common.saving') : t('thr.generate')}
              </Button>
            </div>
          </div>
          {plan ? (
            <div className="mt-4 overflow-x-auto">
              <table className="w-full min-w-[620px] border-collapse text-sm">
                <thead>
                  <tr className="border-b border-[var(--line)] text-left text-xs uppercase tracking-wide text-[var(--kicker)]">
                    <th className="py-2 pr-3">{t('thr.planColEmployee')}</th>
                    <th className="py-2 pr-3">{t('thr.planColService')}</th>
                    <th className="py-2 pr-3 text-right">{t('thr.planColAuto')}</th>
                    <th className="py-2 pr-3 text-right">{t('thr.planColNominal')}</th>
                  </tr>
                </thead>
                <tbody>
                  {plan.items.length === 0 ? (
                    <tr>
                      <td
                        colSpan={4}
                        className="py-4 text-center text-[var(--sea-ink-soft)]"
                      >
                        {t('thr.planEmpty')}
                      </td>
                    </tr>
                  ) : (
                    plan.items.map((item) => (
                      <tr key={item.userId} className="border-b border-[var(--line)]">
                        <td className="py-2 pr-3">
                          <span className="block text-[var(--sea-ink)]">{item.nama}</span>
                          <span className="block font-mono text-xs text-[var(--sea-ink-soft)]">
                            {item.userId}
                          </span>
                        </td>
                        <td className="py-2 pr-3 tabular-nums">{item.masaKerjaTahun}</td>
                        <td className="py-2 pr-3 text-right tabular-nums">
                          {item.nominalOtomatis.toLocaleString('id-ID')}
                        </td>
                        <td className="py-2 pr-3 text-right">
                          <Input
                            type="number"
                            min={0}
                            step={1000}
                            aria-label={`${t('thr.planColNominal')} ${item.nama}`}
                            value={nominals[item.userId] ?? item.nominalOtomatis}
                            onChange={(e) =>
                              setNominals((prev) => ({
                                ...prev,
                                [item.userId]: Number(e.target.value) || 0,
                              }))
                            }
                            className="h-9 w-40 text-right tabular-nums"
                          />
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
              {plan.lewat.length > 0 ? (
                <p className="m-0 mt-2 text-xs text-[var(--sea-ink-soft)]">
                  {t('thr.planSkipped', { count: String(plan.lewat.length) })}
                </p>
              ) : null}
            </div>
          ) : null}
          {report ? (
            <div className="mt-3 rounded-md border border-[var(--chip-line)] bg-[var(--chip-bg)] px-4 py-3 text-sm">
              <p className="m-0 font-semibold text-[var(--sea-ink)]">
                {t('thr.reportTitle', { tahun: String(report.tahun) })}
              </p>
              <p className="m-0 mt-1 tabular-nums text-[var(--sea-ink-soft)]">
                {t('thr.reportStats', {
                  generated: String(report.generated),
                  skipped: String(report.skipped.length),
                })}
              </p>
              {report.skipped.length > 0 ? (
                <ul className="m-0 mt-2 max-h-40 list-disc overflow-y-auto pl-5 text-xs text-[var(--sea-ink-soft)]">
                  {report.skipped.map((skip) => (
                    <li key={skip.userId}>
                      <span className="font-mono">{skip.userId}</span>
                      {' — '}
                      {t(`thr.reason.${skip.reason}` as never)}
                    </li>
                  ))}
                </ul>
              ) : null}
            </div>
          ) : null}
        </section>

        <div className="overflow-x-auto border-t border-[var(--line)] pt-4">
          <table className="w-full min-w-[640px] border-collapse text-sm">
            <thead>
              <tr className="border-b border-[var(--line)] text-left text-xs uppercase tracking-wide text-[var(--kicker)]">
                <th className="py-2 pr-3">{t('thr.colTahun')}</th>
                <th className="py-2 pr-3">{t('thr.colBesaran')}</th>
                <th className="py-2 pr-3">{t('thr.colSyarat')}</th>
                <th className="py-2 pr-3">{t('thr.colBerlaku')}</th>
                <th className="py-2 pr-3 text-right">{t('pesangon.colAksi')}</th>
              </tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-6 text-center text-[var(--sea-ink-soft)]">
                    {t('thr.empty')}
                  </td>
                </tr>
              ) : (
                items.map((item) => (
                  <tr key={item.tahun} className="border-b border-[var(--line)]">
                    <td className="py-2 pr-3 tabular-nums">{item.tahun}</td>
                    <td className="py-2 pr-3">
                      <Badge variant="secondary">
                        {t(`thr.besaran.${item.besaranTipe}` as never)}
                        {item.besaranTipe === 'PERSEN' ||
                        item.besaranTipe === 'PERSEN_TAKE_HOME'
                          ? ` ${item.persen}%`
                          : ''}
                      </Badge>
                    </td>
                    <td className="py-2 pr-3 tabular-nums">
                      {item.syaratMasaKerjaBulan}
                    </td>
                    <td className="py-2 pr-3 tabular-nums">{item.berlakuTanggal}</td>
                    <td className="py-2 pr-3 text-right">
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        onClick={() =>
                          deleteThrConfig(item.tahun)
                            .then(invalidate)
                            .catch(() => toast.error(t('thr.deleteFailed')))
                        }
                      >
                        <Trash2 className="h-4 w-4" aria-hidden="true" />
                        <span className="sr-only">{t('common.delete')}</span>
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
