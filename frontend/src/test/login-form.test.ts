import { createElement } from 'react'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { LocaleProvider } from '@/i18n/locale'
import { LoginForm } from '@/features/auth/login-form'

const mocks = vi.hoisted(() => ({
  signIn: vi.fn(),
  runtimeConfigured: true,
  signInError: null as Error | null,
}))

vi.mock('@/lib/session', () => ({
  isAuthRuntimeConfigured: () => mocks.runtimeConfigured,
  useSignIn: () => ({
    mutateAsync: mocks.signIn,
    isPending: false,
    error: mocks.signInError,
  }),
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  const React = await import('react')
  type MockLinkProps = {
    to: string
    className?: string
    children?: React.ReactNode
  }
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    Link: (props: MockLinkProps) =>
      React.createElement(
        'a',
        { href: props.to, className: props.className },
        props.children,
      ),
  }
})

describe('formulir masuk', () => {
  beforeEach(() => {
    mocks.signIn.mockReset()
    mocks.signIn.mockResolvedValue(undefined)
    mocks.runtimeConfigured = true
    mocks.signInError = null
  })

  it('menampilkan pesan validasi berbahasa Indonesia', async () => {
    render(createElement(LocaleProvider, null, createElement(LoginForm)))

    fireEvent.click(screen.getByRole('button', { name: 'Masuk' }))

    await waitFor(() => {
      expect(screen.getByText('Format email tidak valid')).toBeInTheDocument()
    })
    expect(screen.getByText('Kata sandi wajib diisi')).toBeInTheDocument()
  })

  it('menyerahkan kredensial ke runtime autentikasi', async () => {
    render(createElement(LocaleProvider, null, createElement(LoginForm)))

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'pegawai@instansi.go.id' },
    })
    fireEvent.change(screen.getByLabelText('Kata sandi'), {
      target: { value: 'kata-sandi' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Masuk' }))

    await waitFor(() => {
      expect(mocks.signIn).toHaveBeenCalledWith({
        email: 'pegawai@instansi.go.id',
        password: 'kata-sandi',
      })
    })
  })

  it('menampilkan kegagalan pencatatan masuk dari runtime', () => {
    mocks.signInError = new Error('Email atau kata sandi tidak dikenali.')

    render(createElement(LocaleProvider, null, createElement(LoginForm)))

    expect(screen.getByText(/Email atau kata sandi tidak dikenali\./)).toBeInTheDocument()
  })

  it('menonaktifkan formulir bila runtime autentikasi belum dikonfigurasi', () => {
    mocks.runtimeConfigured = false

    render(createElement(LocaleProvider, null, createElement(LoginForm)))

    expect(screen.getByText(/Layanan autentikasi belum siap/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Masuk' })).toBeDisabled()
    expect(screen.getByLabelText('Email')).toBeDisabled()
  })
})
