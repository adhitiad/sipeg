import { createElement, type ReactNode } from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { LocaleProvider } from '@/i18n/locale'
import { messages, type MessageKey } from '@/i18n/dictionary'
import EmployeeManagementPage from '@/components/EmployeeManagementPage'
import type { EmployeePage } from '@/validations/schemas'

/** Naskah dibaca dari kamus agar tes tetap sah saat susunan kalimat berubah. */
const text = (key: MessageKey) => messages.id[key]

const EMPLOYEE_PAGE: EmployeePage = {
  items: [
    {
      id: '11111111-1111-4111-8111-111111111111',
      nama: 'Siti Rahma',
      email: 'siti.rahma@instansi.go.id',
      nip: '198001012010012001',
      homebaseId: '22222222-2222-4222-8222-222222222222',
      jabatanStruktural: 'Kepala Bagian',
      jabatanFungsional: null,
      aktif: true,
      createdAt: '2026-01-01T00:00:00.000Z',
      updatedAt: '2026-01-01T00:00:00.000Z',
    },
  ],
  page: 1,
  pageSize: 20,
  totalItems: 1,
  totalPages: 1,
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
  },
  query: {
    isPending: false,
    isError: false,
    isFetching: false,
    data: undefined as EmployeePage | undefined,
    refetch: vi.fn(),
  },
  units: {
    data: [
      {
        id: '22222222-2222-4222-8222-222222222222',
        name: 'Bagian Kepegawaian',
        jenis: 'BAGIAN' as const,
        parentId: null,
        aktif: true,
        createdAt: '2026-01-01T00:00:00.000Z',
        updatedAt: '2026-01-01T00:00:00.000Z',
      },
    ],
  },
}))

vi.mock('@/lib/session', () => ({
  useSession: () => mocks.session,
}))

vi.mock('@/lib/employees', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/employees')>()
  return {
    ...actual,
    useEmployees: () => mocks.query,
  }
})

vi.mock('@/lib/organizations', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/organizations')>()
  return {
    ...actual,
    useOrganizations: () => mocks.units,
  }
})

function harness(children: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return createElement(
    QueryClientProvider,
    { client },
    createElement(LocaleProvider, null, children),
  )
}

function resetMocks() {
  mocks.session.isPending = false
  mocks.session.isError = false
  mocks.session.data = {
    userId: 'u-1',
    email: 'operator@instansi.go.id',
    roleName: 'operator',
    permissions: ['employee:read:homebase', 'organization:read:homebase'],
  }
  mocks.query.isPending = false
  mocks.query.isError = false
  mocks.query.isFetching = false
  mocks.query.data = EMPLOYEE_PAGE
  mocks.query.refetch.mockReset()
}

describe('EmployeeManagementPage', () => {
  beforeEach(resetMocks)
  afterEach(() => {
    mocks.query.refetch.mockReset()
  })

  it('menampilkan keadaan tanpa hak akses bila permission baca pegawai tidak ada', () => {
    mocks.session.data = {
      userId: 'u-2',
      email: 'pegawai@instansi.go.id',
      roleName: 'pegawai',
      permissions: [],
    }

    render(harness(createElement(EmployeeManagementPage)))

    expect(screen.getByText(text('state.noAccess.title'))).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('menampilkan radar loader selama daftar dimuat', () => {
    mocks.query.isPending = true
    mocks.query.data = undefined

    render(harness(createElement(EmployeeManagementPage)))

    expect(screen.getByRole('status')).toBeInTheDocument()
    expect(screen.getByText(text('employees.loadingTitle'))).toBeInTheDocument()
  })

  it('menampilkan daftar pegawai beserta nama unitnya', () => {
    render(harness(createElement(EmployeeManagementPage)))

    expect(
      screen.getByRole('heading', { name: text('employees.title') }),
    ).toBeInTheDocument()
    expect(screen.getByText('Siti Rahma')).toBeInTheDocument()
    expect(screen.getByText('Bagian Kepegawaian')).toBeInTheDocument()
    expect(screen.getByText('Kepala Bagian')).toBeInTheDocument()
  })

  it('menampilkan keadaan galat dan mencoba lagi saat permintaan gagal', () => {
    mocks.query.isError = true
    mocks.query.data = undefined

    render(harness(createElement(EmployeeManagementPage)))

    expect(screen.getByText(text('employees.errorTitle'))).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: text('common.retry') }))
    expect(mocks.query.refetch).toHaveBeenCalledTimes(1)
  })

  it('menampilkan keadaan kosong bila tidak ada pegawai dalam cakupan', () => {
    mocks.query.data = { ...EMPLOYEE_PAGE, items: [], totalItems: 0, totalPages: 0 }

    render(harness(createElement(EmployeeManagementPage)))

    expect(screen.getByText(text('employees.emptyTitle'))).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('menyatakan cakupan sesuai peran pemohon', () => {
    render(harness(createElement(EmployeeManagementPage)))

    expect(screen.getByText(text('employees.scopeHomebase'))).toBeInTheDocument()
  })
})
