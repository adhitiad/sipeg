import { createElement, type ReactNode } from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { LocaleProvider } from '@/i18n/locale'
import { messages, type MessageKey } from '@/i18n/dictionary'
import ChangePasswordForm from '@/components/ChangePasswordForm'

const text = (key: MessageKey) => messages.id[key]

const mocks = vi.hoisted(() => ({
  changePassword: vi.fn(),
}))

vi.mock('@/lib/session', () => ({
  changePassword: mocks.changePassword,
}))

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
  mocks.changePassword.mockResolvedValue(undefined)
}

describe('ChangePasswordForm', () => {
  beforeEach(resetMocks)
  afterEach(() => {
    mocks.changePassword.mockReset()
  })

  it('render label dan tombol ubah kata sandi', () => {
    render(harness(createElement(ChangePasswordForm)))

    expect(
      screen.getByLabelText(text('settings.currentPasswordLabel')),
    ).toBeInTheDocument()
    expect(screen.getByLabelText(text('settings.newPasswordLabel'))).toBeInTheDocument()
    expect(
      screen.getByLabelText(text('settings.confirmPasswordLabel')),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: text('settings.changePasswordButton') }),
    ).toBeInTheDocument()
  })

  it('mengirim kata sandi saat disimpan dan memvalidasi cocok', async () => {
    render(harness(createElement(ChangePasswordForm)))

    fireEvent.change(screen.getByLabelText(text('settings.currentPasswordLabel')), {
      target: { value: 'lama1234' },
    })
    fireEvent.change(screen.getByLabelText(text('settings.newPasswordLabel')), {
      target: { value: 'baru1234' },
    })
    fireEvent.change(screen.getByLabelText(text('settings.confirmPasswordLabel')), {
      target: { value: 'salah' },
    })
    fireEvent.click(
      screen.getByRole('button', { name: text('settings.changePasswordButton') }),
    )

    await vi.waitFor(() =>
      expect(screen.getByText(text('settings.passwordMismatch'))).toBeInTheDocument(),
    )
    expect(mocks.changePassword).not.toHaveBeenCalled()
  })

  it('memanggil changePassword bila kata sandi cocok', async () => {
    render(harness(createElement(ChangePasswordForm)))

    fireEvent.change(screen.getByLabelText(text('settings.currentPasswordLabel')), {
      target: { value: 'lama1234' },
    })
    fireEvent.change(screen.getByLabelText(text('settings.newPasswordLabel')), {
      target: { value: 'baru1234' },
    })
    fireEvent.change(screen.getByLabelText(text('settings.confirmPasswordLabel')), {
      target: { value: 'baru1234' },
    })
    fireEvent.click(
      screen.getByRole('button', { name: text('settings.changePasswordButton') }),
    )

    await vi.waitFor(() => expect(mocks.changePassword).toHaveBeenCalledTimes(1))
    const args = mocks.changePassword.mock.calls[0][0]
    expect(args.currentPassword).toBe('lama1234')
    expect(args.newPassword).toBe('baru1234')
    expect(args.confirmPassword).toBe('baru1234')
  })
})
