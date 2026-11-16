import { useState } from 'react'
import { ShieldAlert, Plus } from 'lucide-react'
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
import { fetchKontrak, extendPerpanjangan } from '@/lib/kontrak'

type JenisFilter = '' | 'PKWT' | 'PKWTT'

export default function KontrakPage() {
  const t = useMessages()
  const session = useSession()
  const queryClient = useQueryClient()
  const permissions = session.data?.permissions ?? []
  const canManage = permissions.includes('perpanjangan:manage')
  const canRead =
    permissions.includes('kontrak:read:any') ||
    permissions.includes('kontrak:read:homebase') ||
    permissions.includes('kontrak:read:self')

  const [jenis, setJenis] = useState<JenisFilter>('')
  const [selesaiBefore, setSelesaiBefore] = useState('')
  const [userId, setUserId] = useState('')
  const today = new Date()
  const plus30 = new Date(today.getTime() + 30 * 24 * 60 * 60 * 1000)
  const defaultPlus30 = plus30.toISOString().slice(0, 10)

  const query = useQuery({
    queryKey: ['kontrak', jenis, selesaiBefore, userId] as const,
    queryFn: () =>
      fetchKontrak({
        jenis: jenis || undefined,
        selesaiBefore: selesaiBefore || undefined,
        userId: userId.trim() || undefined,
      }),
    enabled: canRead,
    staleTime: 15_000,
    retry: false,
  })

  const [extendOpen, setExtendOpen] = useState(false)
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['kontrak'] })

  const items = query.data?.items ?? []
  const expiringSoon = items.filter((k) => k.selesai && k.selesai <= defaultPlus30)

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
      <p className="island-kicker mb-2">{t('kontrak.kicker')}</p>
      <h1 className="display-title mb-2 text-3xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('kontrak.title')}
      </h1>
      <p className="m-0 mb-6 max-w-3xl text-base leading-relaxed text-[var(--sea-ink-soft)]">
        {t('kontrak.lead')}
      </p>

      <div className="island-shell flex flex-col gap-4 rounded-2xl p-4 sm:p-6">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div className="flex flex-wrap gap-2">
            {(['', 'PKWT', 'PKWTT'] as const).map((value) => (
              <button
                key={value || 'ALL'}
                type="button"
                onClick={() => setJenis(value as JenisFilter)}
                aria-pressed={jenis === value}
                className={`rounded-md border px-3 py-1.5 text-xs font-semibold uppercase tracking-wide ${
                  jenis === value
                    ? 'border-[var(--sea-ink)] bg-[var(--sea-ink)] text-[var(--paper,#fff)]'
                    : 'border-[var(--chip-line)] bg-[var(--chip-bg)] text-[var(--sea-ink)]'
                }`}
              >
                {value || t('kontrak.filterAll')}
              </button>
            ))}
            <Input
              type="date"
              value={selesaiBefore}
              onChange={(e) => setSelesaiBefore(e.target.value)}
              className="h-11 w-44"
              aria-label={t('kontrak.filterSelesaiBefore')}
            />
            <Input
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              placeholder={t('kontrak.filterUserId')}
              className="h-11 w-64"
            />
          </div>
          {canManage ? (
            <Button type="button" className="gap-2" onClick={() => setExtendOpen(true)}>
              <Plus className="h-4 w-4" aria-hidden="true" />
              {t('kontrak.extend')}
            </Button>
          ) : null}
        </div>

        {expiringSoon.length > 0 ? (
          <p
            className="m-0 rounded-md border border-amber-400/60 bg-amber-50 px-3 py-2 text-sm text-amber-900"
            role="status"
          >
            {t('kontrak.expiringSoon', { count: String(expiringSoon.length) })}
          </p>
        ) : null}

        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] border-collapse text-sm">
            <thead>
              <tr className="border-b border-[var(--line)] text-left text-xs uppercase tracking-wide text-[var(--kicker)]">
                <th className="py-2 pr-3">{t('kontrak.colMulai')}</th>
                <th className="py-2 pr-3">{t('kontrak.colSelesai')}</th>
                <th className="py-2 pr-3">{t('kontrak.colJenis')}</th>
                <th className="py-2 pr-3">{t('kontrak.colStatus')}</th>
                <th className="py-2 pr-3">{t('kontrak.colNomorSk')}</th>
              </tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-6 text-center text-[var(--sea-ink-soft)]">
                    {t('kontrak.empty')}
                  </td>
                </tr>
              ) : (
                items.map((k) => {
                  const soon = k.selesai && k.selesai <= defaultPlus30
                  return (
                    <tr key={k.id} className="border-b border-[var(--line)]">
                      <td className="py-2 pr-3 tabular-nums">{k.mulai}</td>
                      <td className="py-2 pr-3 tabular-nums">{k.selesai ?? '—'}</td>
                      <td className="py-2 pr-3">{k.jenis}</td>
                      <td className="py-2 pr-3">
                        <Badge variant={soon ? 'destructive' : 'secondary'}>
                          {k.status}
                        </Badge>
                      </td>
                      <td className="py-2 pr-3 font-mono text-xs">{k.nomorSk}</td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {extendOpen ? (
        <ExtendDialog
          onClose={() => setExtendOpen(false)}
          onSuccess={() => {
            setExtendOpen(false)
            invalidate()
          }}
        />
      ) : null}
    </div>
  )
}

function ExtendDialog({
  onClose,
  onSuccess,
}: {
  onClose: () => void
  onSuccess: () => void
}) {
  const t = useMessages()
  const [userId, setUserId] = useState('')
  const [durasi, setDurasi] = useState<number | ''>(12)
  const [allowOverlap, setAllowOverlap] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async () => {
    setSubmitting(true)
    try {
      await extendPerpanjangan({
        userId,
        durasiBulan: durasi === '' ? null : Number(durasi) || null,
        allowOverlap,
      })
      toast.success(t('kontrak.extendSuccess'))
      onSuccess()
    } catch {
      toast.error(t('kontrak.extendError'))
      setSubmitting(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle>{t('kontrak.extendTitle')}</DialogTitle>
          <DialogDescription>{t('kontrak.extendDescription')}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-3">
          <div>
            <Label htmlFor="userId">{t('kontrak.fieldUserId')}</Label>
            <Input
              id="userId"
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              placeholder="UUID pegawai"
              className="h-11"
            />
          </div>
          <div>
            <Label htmlFor="durasi">{t('kontrak.fieldDurasi')}</Label>
            <Input
              id="durasi"
              type="number"
              min={1}
              max={60}
              value={durasi}
              onChange={(e) =>
                setDurasi(e.target.value === '' ? '' : Number(e.target.value))
              }
              className="h-11"
            />
          </div>
          <div className="flex items-start gap-2">
            <input
              id="allowOverlap"
              type="checkbox"
              checked={allowOverlap}
              onChange={(e) => setAllowOverlap(e.target.checked)}
              className="mt-1 h-4 w-4 rounded border-[var(--chip-line)] accent-[var(--sea-ink)]"
            />
            <label htmlFor="allowOverlap" className="text-sm text-[var(--sea-ink)]">
              {t('kontrak.allowOverlap')}
              <span className="block text-xs text-[var(--sea-ink-soft)]">
                {t('kontrak.allowOverlapHint')}
              </span>
            </label>
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            type="button"
            disabled={submitting || !userId || durasi === ''}
            onClick={handleSubmit}
          >
            {submitting ? t('common.saving') : t('common.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
