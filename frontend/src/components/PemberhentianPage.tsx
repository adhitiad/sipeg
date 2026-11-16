import { useState } from 'react'
import { ShieldAlert, Plus, Eye } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { toast } from 'sonner'
import { useMessages } from '@/i18n/locale-context'
import { useSession } from '@/lib/session'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchPemberhentian,
  fetchKompensasiPreview,
  proposePemberhentian,
  approvePemberhentian,
  rejectPemberhentian,
} from '@/lib/pemberhentian'

export default function PemberhentianPage() {
  const t = useMessages()
  const session = useSession()
  const queryClient = useQueryClient()
  const permissions = session.data?.permissions ?? []
  const canPropose =
    permissions.includes('pemberhentian:propose:any') ||
    permissions.includes('pemberhentian:propose:homebase')
  const canApprove = permissions.includes('pemberhentian:approve')
  const canRead =
    permissions.includes('pemberhentian:read:any') ||
    permissions.includes('pemberhentian:read:homebase')

  const [status, setStatus] = useState<string>('MENUNGGU')
  const [userIdFilter, setUserIdFilter] = useState('')

  const query = useQuery({
    queryKey: ['pemberhentian', status, userIdFilter] as const,
    queryFn: () =>
      fetchPemberhentian({
        status: status || undefined,
        userId: userIdFilter.trim() || undefined,
      }),
    enabled: canRead,
    staleTime: 10_000,
    retry: false,
  })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['pemberhentian'] })
    void queryClient.invalidateQueries({ queryKey: ['pemberhentian', 'preview'] })
  }

  const [proposeOpen, setProposeOpen] = useState(false)
  const [active, setActive] = useState<
    import('@/validations/schemas').Pemberhentian | null
  >(null)
  const [action, setAction] = useState<'approve' | 'reject' | 'detail' | null>(null)

  const items = query.data?.items ?? []

  if (session.data && !canRead) {
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
      <p className="island-kicker mb-2">{t('pemberhentian.kicker')}</p>
      <h1 className="display-title mb-2 text-3xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('pemberhentian.title')}
      </h1>
      <p className="m-0 mb-6 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)]">
        {t('pemberhentian.lead')}
      </p>

      <div className="island-shell flex flex-col gap-4 rounded-2xl p-4 sm:p-6">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div
            className="flex flex-wrap gap-2"
            role="radiogroup"
            aria-label={t('pemberhentian.filterByStatus')}
          >
            {(['MENUNGGU', 'DISETUJUI', 'DITOLAK'] as const).map((value) => (
              <button
                key={value}
                type="button"
                onClick={() => setStatus(value)}
                aria-pressed={status === value}
                className={`rounded-md border px-3 py-1.5 text-xs font-semibold uppercase tracking-wide ${
                  status === value
                    ? 'border-[var(--sea-ink)] bg-[var(--sea-ink)] text-[var(--paper,#fff)]'
                    : 'border-[var(--chip-line)] bg-[var(--chip-bg)] text-[var(--sea-ink)]'
                }`}
              >
                {t(`pemberhentian.status.${value}` as never)}
              </button>
            ))}
          </div>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <Label
              htmlFor="user-id"
              className="text-xs font-semibold uppercase tracking-wide text-[var(--kicker)]"
            >
              {t('pemberhentian.filterByUser')}
            </Label>
            <Input
              id="user-id"
              value={userIdFilter}
              onChange={(e) => setUserIdFilter(e.target.value)}
              placeholder="UUID pegawai"
              className="h-11 w-64"
            />
            {canPropose ? (
              <Button
                type="button"
                className="gap-2"
                onClick={() => setProposeOpen(true)}
              >
                <Plus className="h-4 w-4" aria-hidden="true" />
                {t('pemberhentian.propose')}
              </Button>
            ) : null}
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full min-w-[720px] border-collapse text-sm">
            <thead>
              <tr className="border-b border-[var(--line)] text-left text-xs uppercase tracking-wide text-[var(--kicker)]">
                <th className="py-2 pr-3">{t('pemberhentian.colJenis')}</th>
                <th className="py-2 pr-3">{t('pemberhentian.colAlasan')}</th>
                <th className="py-2 pr-3">{t('pemberhentian.colTanggal')}</th>
                <th className="py-2 pr-3">{t('pemberhentian.colStatus')}</th>
                <th className="py-2 pr-3">{t('pemberhentian.colNomorSk')}</th>
                <th className="py-2 pr-3 text-right">
                  {t('pemberhentian.colKompensasi')}
                </th>
                <th className="py-2 pr-3 text-right">{t('pemberhentian.colAksi')}</th>
              </tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-6 text-center text-[var(--sea-ink-soft)]">
                    {t('pemberhentian.empty')}
                  </td>
                </tr>
              ) : (
                items.map((item) => (
                  <tr key={item.id} className="border-b border-[var(--line)]">
                    <td className="py-2 pr-3 text-xs">{item.jenis}</td>
                    <td className="py-2 pr-3">{item.alasan}</td>
                    <td className="py-2 pr-3 tabular-nums">{item.tanggal}</td>
                    <td className="py-2 pr-3">
                      <Badge
                        variant={item.status === 'DISETUJUI' ? 'default' : 'secondary'}
                      >
                        {item.status}
                      </Badge>
                    </td>
                    <td className="py-2 pr-3 font-mono text-xs">{item.nomorSk || '—'}</td>
                    <td className="py-2 pr-3 text-right tabular-nums">
                      {item.kompensasiBulan > 0 ? String(item.kompensasiBulan) : '—'}
                    </td>
                    <td className="py-2 pr-3 text-right">
                      <div className="inline-flex gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            setActive(item)
                            setAction('detail')
                          }}
                          aria-label={t('common.detail')}
                        >
                          <Eye className="h-4 w-4" aria-hidden="true" />
                        </Button>
                        {canApprove && item.status === 'MENUNGGU' ? (
                          <>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => {
                                setActive(item)
                                setAction('approve')
                              }}
                            >
                              {t('common.approve')}
                            </Button>
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => {
                                setActive(item)
                                setAction('reject')
                              }}
                            >
                              {t('common.reject')}
                            </Button>
                          </>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {proposeOpen ? (
        <ProposeDialog
          onClose={() => setProposeOpen(false)}
          onSuccess={() => {
            setProposeOpen(false)
            invalidate()
          }}
        />
      ) : null}
      {active && action === 'detail' ? (
        <DetailDialog
          item={active}
          onClose={() => {
            setActive(null)
            setAction(null)
          }}
        />
      ) : null}
      {active && action && action !== 'detail' ? (
        <DecisionDialog
          item={active}
          action={action}
          onClose={() => {
            setActive(null)
            setAction(null)
          }}
          onSuccess={() => {
            setActive(null)
            setAction(null)
            invalidate()
          }}
        />
      ) : null}
    </div>
  )
}

function ProposeDialog({
  onClose,
  onSuccess,
}: {
  onClose: () => void
  onSuccess: () => void
}) {
  const t = useMessages()
  const [userId, setUserId] = useState('')
  const [alasan, setAlasan] = useState('BUP')
  const [tanggal, setTanggal] = useState('')
  const [keterangan, setKeterangan] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async () => {
    setSubmitting(true)
    try {
      await proposePemberhentian({
        userId,
        alasan: alasan as never,
        tanggal,
        keterangan: keterangan.trim() || null,
      })
      toast.success(t('pemberhentian.proposeSuccess'))
      onSuccess()
    } catch {
      toast.error(t('pemberhentian.proposeError'))
      setSubmitting(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[520px]">
        <DialogHeader>
          <DialogTitle>{t('pemberhentian.proposeTitle')}</DialogTitle>
          <DialogDescription>{t('pemberhentian.proposeDescription')}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-3">
          <div>
            <Label htmlFor="userId">{t('pemberhentian.fieldUserId')}</Label>
            <Input
              id="userId"
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              placeholder="UUID pegawai"
              className="h-11"
            />
          </div>
          <div>
            <Label htmlFor="alasan">{t('pemberhentian.fieldAlasan')}</Label>
            <select
              id="alasan"
              value={alasan}
              onChange={(e) => setAlasan(e.target.value)}
              className="flex h-11 w-full rounded-md border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 text-sm text-[var(--sea-ink)]"
            >
              {(
                [
                  'MENINGGAL_DUNIA',
                  'SAKIT_BERKEPANJANGAN',
                  'BUP',
                  'MENGUNDUR_DIRI',
                  'PENYEDERHANAAN_ORGANISASI',
                  'KESALAHAN_BERAT',
                ] as const
              ).map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
          </div>
          <div>
            <Label htmlFor="tanggal">{t('pemberhentian.fieldTanggal')}</Label>
            <Input
              id="tanggal"
              type="date"
              value={tanggal}
              onChange={(e) => setTanggal(e.target.value)}
              className="h-11"
            />
          </div>
          <div>
            <Label htmlFor="keterangan">{t('pemberhentian.fieldKeterangan')}</Label>
            <Input
              id="keterangan"
              value={keterangan}
              onChange={(e) => setKeterangan(e.target.value)}
              className="h-11"
            />
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            type="button"
            disabled={submitting || !userId || !tanggal}
            onClick={handleSubmit}
          >
            {submitting ? t('common.saving') : t('common.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function DecisionDialog({
  item,
  action,
  onClose,
  onSuccess,
}: {
  item: import('@/validations/schemas').Pemberhentian
  action: 'approve' | 'reject'
  onClose: () => void
  onSuccess: () => void
}) {
  const t = useMessages()
  const [note, setNote] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const preview = useQuery({
    queryKey: ['pemberhentian', 'preview', item.id] as const,
    queryFn: () => fetchKompensasiPreview(item.id),
    enabled: action === 'approve',
    staleTime: 30_000,
    retry: false,
  })

  const handleSubmit = async () => {
    setSubmitting(true)
    try {
      if (action === 'approve') {
        await approvePemberhentian(item.id, note.trim() || undefined)
      } else {
        await rejectPemberhentian(item.id, note.trim() || undefined)
      }
      toast.success(
        t(
          action === 'approve'
            ? 'pemberhentian.approveSuccess'
            : 'pemberhentian.rejectSuccess',
        ),
      )
      onSuccess()
    } catch {
      toast.error(
        t(
          action === 'approve'
            ? 'pemberhentian.approveError'
            : 'pemberhentian.rejectError',
        ),
      )
      setSubmitting(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle>
            {action === 'approve'
              ? t('pemberhentian.approveTitle')
              : t('pemberhentian.rejectTitle')}
          </DialogTitle>
          <DialogDescription>
            {t('pemberhentian.decisionDescription', { alasan: item.alasan })}
          </DialogDescription>
        </DialogHeader>
        {action === 'approve' ? (
          <div
            className="rounded-md border border-[var(--chip-line)] bg-[var(--chip-bg)] px-4 py-3 text-sm"
            aria-live="polite"
          >
            <p className="mb-1 text-xs font-semibold uppercase tracking-[0.16em] text-[var(--kicker)]">
              {t('pemberhentian.previewTitle')}
            </p>
            {preview.isPending ? (
              <p className="m-0 text-xs text-[var(--sea-ink-soft)]">
                {t('common.loading')}
              </p>
            ) : preview.isError ? (
              <p className="m-0 text-xs text-[var(--sea-ink-soft)]">
                {t('pemberhentian.previewError')}
              </p>
            ) : preview.data?.honoured === false ? (
              <p className="m-0 text-xs text-[var(--sea-ink-soft)]">
                {t('pemberhentian.previewTidakHormat')}
              </p>
            ) : preview.data?.tersedia === false ? (
              <p className="m-0 text-xs text-[var(--sea-ink-soft)]">
                {t('pemberhentian.previewTidakTersedia')}
              </p>
            ) : preview.data ? (
              <dl className="m-0 grid grid-cols-2 gap-x-6 gap-y-1">
                <div className="flex items-baseline justify-between gap-3">
                  <dt className="text-xs text-[var(--sea-ink-soft)]">
                    {t('pemberhentian.previewMasaKerja')}
                  </dt>
                  <dd className="m-0 tabular-nums font-semibold text-[var(--sea-ink)]">
                    {preview.data.masaKerjaTahun}
                  </dd>
                </div>
                <div className="flex items-baseline justify-between gap-3">
                  <dt className="text-xs text-[var(--sea-ink-soft)]">
                    {t('pemberhentian.previewPesangon')}
                  </dt>
                  <dd className="m-0 tabular-nums font-semibold text-[var(--sea-ink)]">
                    {preview.data.pesangonBulan}
                  </dd>
                </div>
                <div className="flex items-baseline justify-between gap-3">
                  <dt className="text-xs text-[var(--sea-ink-soft)]">
                    {t('pemberhentian.previewTambahan')}
                  </dt>
                  <dd className="m-0 tabular-nums font-semibold text-[var(--sea-ink)]">
                    {preview.data.tambahanBulan}
                  </dd>
                </div>
                <div className="flex items-baseline justify-between gap-3">
                  <dt className="text-xs font-semibold text-[var(--kicker)]">
                    {t('pemberhentian.previewTotal')}
                  </dt>
                  <dd className="m-0 tabular-nums text-base font-bold text-[var(--sea-ink)]">
                    {preview.data.totalBulan}
                  </dd>
                </div>
              </dl>
            ) : null}
          </div>
        ) : null}
        <div>
          <Label htmlFor="note">{t('pemberhentian.fieldNote')}</Label>
          <Input
            id="note"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            className="h-11"
          />
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            type="button"
            disabled={submitting}
            variant={action === 'approve' ? 'default' : 'destructive'}
            onClick={handleSubmit}
          >
            {submitting
              ? t('common.saving')
              : action === 'approve'
                ? t('common.approve')
                : t('common.reject')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function DetailDialog({
  item,
  onClose,
}: {
  item: import('@/validations/schemas').Pemberhentian
  onClose: () => void
}) {
  const t = useMessages()
  const rows: Array<[string, string]> = [
    [t('pemberhentian.colJenis'), item.jenis],
    [t('pemberhentian.colAlasan'), item.alasan],
    [t('pemberhentian.colTanggal'), item.tanggal],
    [t('pemberhentian.colStatus'), item.status],
    [t('pemberhentian.colNomorSk'), item.nomorSk || '—'],
    [t('pemberhentian.colHakTerakhir'), item.hakTerakhir ?? '—'],
    [t('pemberhentian.colKompensasi'), String(item.kompensasiBulan)],
    [t('pemberhentian.fieldKeterangan'), item.keterangan ?? '—'],
    [t('pemberhentian.colCreatedAt'), item.createdAt],
  ]

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>{t('pemberhentian.detailTitle')}</DialogTitle>
          <DialogDescription>
            {t('pemberhentian.detailDescription', { userId: item.userId })}
          </DialogDescription>
        </DialogHeader>
        <dl className="grid grid-cols-1 gap-x-6 gap-y-3 border-t border-[var(--line)] pt-4 sm:grid-cols-2">
          {rows.map(([label, value]) => (
            <div key={label}>
              <dt className="text-xs font-semibold uppercase tracking-wide text-[var(--kicker)]">
                {label}
              </dt>
              <dd className="m-0 mt-0.5 text-sm text-[var(--sea-ink)]">{value}</dd>
            </div>
          ))}
        </dl>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t('common.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
