import { createElement, type ReactNode } from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { SloMetrics } from '@/validations/schemas'
import { LocaleProvider } from '@/i18n/locale'
import SloMetricsPage from '@/components/SloMetricsPage'

const EMPTY_METRICS: SloMetrics = {
  windowStart: '2026-11-13T00:00:00.000Z',
  windowEnd: '2026-11-13T01:00:00.000Z',
  requests: 0,
  errors: 0,
  errorRate: 0,
  avgMs: 0,
  p95Ms: 0,
  p99Ms: 0,
  maxMs: 0,
  routes: [],
}

const FULL_METRICS: SloMetrics = {
  windowStart: '2026-11-13T00:00:00.000Z',
  windowEnd: '2026-11-13T01:00:00.000Z',
  requests: 1280,
  errors: 12,
  errorRate: 0.0094,
  avgMs: 41.2,
  p95Ms: 820,
  p99Ms: 1300,
  maxMs: 2100,
  routes: [
    {
      route: '/api/v1/employees',
      method: 'GET',
      requests: 1280,
      errors: 12,
      errorRate: 0.0094,
      avgMs: 41.2,
      p95Ms: 820,
      p99Ms: 1300,
      maxMs: 2100,
    },
  ],
}

const mocks = vi.hoisted(() => ({
  session: {
    isPending: false,
    isError: false,
    data: null as {
      userId: string
      email: string
      roleName: string
      permissions: string[]
    } | null,
    refetch: vi.fn(),
  },
  query: {
    isPending: false,
    isError: false,
    data: undefined as SloMetrics | undefined,
    refetch: vi.fn(),
  },
}))

vi.mock('@/lib/session', () => ({
  useSession: () => mocks.session,
}))

vi.mock('@/lib/slo', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/slo')>()
  return {
    ...actual,
    useSloMetrics: () => mocks.query,
  }
})

function harness(children: ReactNode) {
  return createElement(LocaleProvider, null, children)
}

function resetMocks() {
  mocks.session.isPending = false
  mocks.session.isError = false
  mocks.session.data = {
    userId: 'u-1',
    email: 'admin@instansi.go.id',
    roleName: 'admin',
    permissions: ['slo:read'],
  }
  mocks.query.isPending = false
  mocks.query.isError = false
  mocks.query.data = FULL_METRICS
  mocks.query.refetch.mockReset()
}

describe('SloMetricsPage', () => {
  beforeEach(resetMocks)
  afterEach(() => {
    mocks.query.refetch.mockReset()
  })

  it('menampilkan keadaan tanpa hak akses bila permission slo:read tidak ada', () => {
    mocks.session.data = {
      userId: 'u-2',
      email: 'operator@instansi.go.id',
      roleName: 'operator',
      permissions: ['audit:read:homebase'],
    }

    render(harness(createElement(SloMetricsPage)))

    expect(screen.getByText('Belum ada hak akses')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('menampilkan keadaan memuat selama kueri berjalan', () => {
    mocks.query.isPending = true
    mocks.query.data = undefined

    render(harness(createElement(SloMetricsPage)))

    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('menampilkan keadaan galat dan memanggil refetch saat tombol dicoba lagi', () => {
    mocks.query.isError = true
    mocks.query.data = undefined

    render(harness(createElement(SloMetricsPage)))

    expect(screen.getByText('Metrik tidak dapat dimuat')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Coba lagi' }))
    expect(mocks.query.refetch).toHaveBeenCalledTimes(1)
  })

  it('menampilkan keadaan kosong bila belum ada metrik sama sekali', () => {
    mocks.query.data = EMPTY_METRICS

    render(harness(createElement(SloMetricsPage)))

    expect(screen.getByText('Belum ada metrik terkumpul')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('menampilkan total dan tabel per rute saat data tersedia', () => {
    render(harness(createElement(SloMetricsPage)))

    expect(screen.getByText('/api/v1/employees')).toBeInTheDocument()
    expect(screen.getAllByText('1.280').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByRole('table', { name: /Metrik SLO per rute/ })).toBeInTheDocument()
  })
})
