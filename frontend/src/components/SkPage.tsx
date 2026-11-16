import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Eye, ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useMessages } from '@/i18n/locale-context'
import { useSession } from '@/lib/session'
import { fetchSkList } from '@/lib/sk'
import type { MessageKey } from '@/i18n/dictionary'
import type { Sk } from '@/validations/schemas'
import LoadingRadar from '@/components/LoadingRadar'
import RetryButton from '@/components/RetryButton'

const JENIS = [
  'PENGANGKATAN',
  'MUTASI',
  'PHK',
  'JABATAN_STRUKTURAL',
  'PENUGASAN',
] as const
const PENERBIT = ['YAYASAN', 'REKTOR'] as const

const JENIS_LABEL: Record<string, MessageKey> = {
  PENGANGKATAN: 'sk.jenis.PENGANGKATAN',
  MUTASI: 'sk.jenis.MUTASI',
  PHK: 'sk.jenis.PHK',
  JABATAN_STRUKTURAL: 'sk.jenis.JABATAN_STRUKTURAL',
  PENUGASAN: 'sk.jenis.PENUGASAN',
}

const PENERBIT_LABEL: Record<string, MessageKey> = {
  YAYASAN: 'sk.penerbit.YAYASAN',
  REKTOR: 'sk.penerbit.REKTOR',
}

function jenisLabel(jenis: string): MessageKey {
  return JENIS_LABEL[jenis] ?? 'sk.jenis'
}

function penerbitLabel(penerbit: string): MessageKey {
  return PENERBIT_LABEL[penerbit] ?? 'sk.penerbit'
}

export default function SkPage() {
  const t = useMessages()
  const session = useSession()
  const permissions = session.data?.permissions ?? []
  const canRead =
    permissions.includes('sk:read:any') || permissions.includes('sk:read:homebase')

  const [search, setSearch] = useState('')
  const [jenis, setJenis] = useState('')
  const [penerbit, setPenerbit] = useState('')
  const [active, setActive] = useState<Sk | null>(null)

  const query = useQuery({
    queryKey: ['sk', 'list', search, jenis, penerbit] as const,
    queryFn: () =>
      fetchSkList({
        search: search.trim() || undefined,
        jenis: jenis || undefined,
        penerbit: penerbit || undefined,
        pageSize: 50,
      }),
    enabled: canRead,
    staleTime: 30_000,
    retry: false,
  })

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

  const items = query.data?.items ?? []

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('sk.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('sk.title')}
      </h1>
      <p className="mb-6 max-w-3xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('sk.lead')}
      </p>

      <section className="island-shell rounded-2xl p-5">
        <div className="mb-4 grid gap-2 lg:grid-cols-3">
          <div>
            <Label htmlFor="sk-search">{t('sk.search')}</Label>
            <Input
              id="sk-search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('sk.searchPlaceholder')}
            />
          </div>
          <div>
            <Label htmlFor="sk-jenis">{t('sk.jenis')}</Label>
            <Select value={jenis} onValueChange={setJenis}>
              <SelectTrigger id="sk-jenis" className="mt-1 w-full">
                <SelectValue placeholder={t('sk.jenis')} />
              </SelectTrigger>
              <SelectContent>
                {JENIS.map((item) => (
                  <SelectItem key={item} value={item}>
                    {t(jenisLabel(item))}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label htmlFor="sk-penerbit">{t('sk.penerbit')}</Label>
            <Select value={penerbit} onValueChange={setPenerbit}>
              <SelectTrigger id="sk-penerbit" className="mt-1 w-full">
                <SelectValue placeholder={t('sk.penerbit')} />
              </SelectTrigger>
              <SelectContent>
                {PENERBIT.map((item) => (
                  <SelectItem key={item} value={item}>
                    {t(penerbitLabel(item))}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        {query.isPending ? (
          <div className="flex flex-col items-center gap-3 py-8">
            <LoadingRadar size="md" decorative />
            <p className="m-0 text-sm text-[var(--sea-ink-soft)]">{t('sk.loading')}</p>
          </div>
        ) : query.isError ? (
          <div className="py-6">
            <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
              {t('sk.errorTitle')}
            </h2>
            <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
              {t('sk.errorDescription')}
            </p>
            <RetryButton label={t('common.retry')} onClick={() => void query.refetch()} />
          </div>
        ) : items.length === 0 ? (
          <p className="text-sm text-[var(--sea-ink-soft)]">{t('sk.empty')}</p>
        ) : (
          <div className="space-y-2">
            {items.map((item) => (
              <div
                key={item.id}
                className="flex items-center justify-between gap-2 rounded-lg border border-[var(--line)] p-2 text-sm"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{item.nomorSk}</span>
                  <Badge variant="outline">{t(jenisLabel(item.jenis))}</Badge>
                  <span className="text-[var(--sea-ink-soft)]">
                    {t(penerbitLabel(item.penerbit))}
                  </span>
                  <span className="text-[var(--sea-ink-soft)]">
                    {item.nama ?? item.userId}
                  </span>
                  <span className="text-[var(--sea-ink-soft)]">{item.tanggalSk}</span>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setActive(item)}
                  aria-label={t('sk.detail')}
                >
                  <Eye className="h-4 w-4" aria-hidden="true" />
                </Button>
              </div>
            ))}
          </div>
        )}
      </section>

      <Dialog open={active !== null} onOpenChange={(open) => !open && setActive(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{active?.nomorSk ?? t('sk.detail')}</DialogTitle>
            <DialogDescription>{t('sk.detailDescription')}</DialogDescription>
          </DialogHeader>
          {active && (
            <dl className="m-0 grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
              <dt className="text-[var(--sea-ink-soft)]">{t('sk.jenis')}</dt>
              <dd className="m-0">{t(jenisLabel(active.jenis))}</dd>
              <dt className="text-[var(--sea-ink-soft)]">{t('sk.penerbit')}</dt>
              <dd className="m-0">{t(penerbitLabel(active.penerbit))}</dd>
              <dt className="text-[var(--sea-ink-soft)]">{t('sk.colPegawai')}</dt>
              <dd className="m-0">{active.nama ?? active.userId}</dd>
              <dt className="text-[var(--sea-ink-soft)]">{t('sk.tanggalSk')}</dt>
              <dd className="m-0">{active.tanggalSk}</dd>
              <dt className="text-[var(--sea-ink-soft)]">{t('sk.mulaiBerlaku')}</dt>
              <dd className="m-0">{active.mulaiBerlaku ?? '-'}</dd>
              <dt className="text-[var(--sea-ink-soft)]">{t('sk.selesaiBerlaku')}</dt>
              <dd className="m-0">{active.selesaiBerlaku ?? '-'}</dd>
              {active.metadata ? (
                <>
                  <dt className="text-[var(--sea-ink-soft)]">{t('sk.metadata')}</dt>
                  <dd className="m-0 break-all text-xs">{active.metadata}</dd>
                </>
              ) : null}
            </dl>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
