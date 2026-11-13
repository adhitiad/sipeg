import { createElement, type ReactNode } from 'react'
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LocaleProvider } from '@/i18n/locale'
import { RequireAuth, RequireNoAuth } from '@/lib/require-auth'

const mocks = vi.hoisted(() => ({
  session: {
    isPending: false,
    isError: false,
    data: null as { userId: string; email: string } | null,
    error: null as Error | null,
    refetch: vi.fn(),
  },
}))

vi.mock('@/lib/session', () => ({
  useSession: () => mocks.session,
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Navigate: ({ to }: { to: string }) =>
      createElement('div', { 'data-testid': 'redirect' }, to),
  }
})

function page(text: string) {
  return createElement('div', null, text)
}

function withLocale(children: ReactNode) {
  return createElement(LocaleProvider, null, children)
}

function harness(children: ReactNode) {
  return withLocale(createElement(RequireAuth, null, children))
}

function resetSession() {
  mocks.session.isPending = false
  mocks.session.isError = false
  mocks.session.data = null
  mocks.session.error = null
  mocks.session.refetch.mockReset()
}

describe('RequireAuth', () => {
  afterEach(resetSession)

  it('menampilkan keadaan memuat saat sesi belum diperiksa', () => {
    mocks.session.isPending = true

    render(harness(page('halaman beranda')))

    expect(screen.getByText('Memuat sesi Anda...')).toBeInTheDocument()
  })

  it('menampilkan keadaan gagal dan tombol coba lagi', () => {
    mocks.session.isError = true
    mocks.session.error = new Error('gagal')

    render(harness(page('halaman beranda')))

    expect(screen.getByText('Sesi tidak dapat diperiksa')).toBeInTheDocument()
    screen.getByRole('button', { name: 'Coba lagi' }).click()
    expect(mocks.session.refetch).toHaveBeenCalled()
  })

  it('mengarahkan tamu ke halaman masuk', () => {
    render(harness(page('halaman beranda')))

    expect(screen.getByTestId('redirect')).toHaveTextContent('/login')
    expect(screen.queryByText('halaman beranda')).not.toBeInTheDocument()
  })

  it('menampilkan isi halaman bila sesi aktif', () => {
    mocks.session.data = { userId: 'u-1', email: 'pegawai@instansi.go.id' }

    render(harness(page('halaman beranda')))

    expect(screen.getByText('halaman beranda')).toBeInTheDocument()
  })
})

describe('RequireNoAuth', () => {
  afterEach(resetSession)

  it('mengarahkan pengguna yang sudah masuk ke ruang kerja', () => {
    mocks.session.data = { userId: 'u-1', email: 'pegawai@instansi.go.id' }

    render(withLocale(createElement(RequireNoAuth, null, page('halaman masuk'))))

    expect(screen.getByTestId('redirect')).toHaveTextContent('/workspace')
    expect(screen.queryByText('halaman masuk')).not.toBeInTheDocument()
  })

  it('menampilkan halaman masuk bila belum ada sesi', () => {
    render(withLocale(createElement(RequireNoAuth, null, page('halaman masuk'))))

    expect(screen.getByText('halaman masuk')).toBeInTheDocument()
  })
})
