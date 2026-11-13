import { createElement, type ReactNode } from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { LocaleProvider } from '@/i18n/locale'
import { messages, type MessageKey } from '@/i18n/dictionary'
import ProfileForm from '@/components/ProfileForm'

const text = (key: MessageKey) => messages.id[key]

const mocks = vi.hoisted(() => ({
  updateProfile: vi.fn(),
}))

vi.mock('@/lib/session', () => ({
  useUpdateProfile: () => ({
    mutateAsync: mocks.updateProfile,
    isPending: false,
  }),
}))

const ACCOUNT = {
  userId: '11111111-1111-4111-8111-111111111111',
  email: 'pegawai@instansi.go.id',
  roleName: 'pegawai',
  homebaseId: '22222222-2222-4222-8222-222222222222',
  permissions: ['payroll:read:self'],
}

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
  mocks.updateProfile.mockResolvedValue({
    ...ACCOUNT,
    nama: 'Budi Santoso',
    noHp: '081234567890',
  })
}

describe('ProfileForm', () => {
  beforeEach(resetMocks)
  afterEach(() => {
    mocks.updateProfile.mockReset()
  })

  it('render label dan input profil', () => {
    render(harness(createElement(ProfileForm)))

    expect(screen.getByLabelText(text('settings.profileNameLabel'))).toBeInTheDocument()
    expect(screen.getByLabelText(text('settings.noHpLabel'))).toBeInTheDocument()
    expect(screen.getByRole('button', { name: text('common.save') })).toBeInTheDocument()
  })

  it('mengirim nama dan nomor HP yang sudah diformat saat disimpan', async () => {
    render(harness(createElement(ProfileForm)))

    fireEvent.change(screen.getByLabelText(text('settings.profileNameLabel')), {
      target: { value: '  Budi   Santoso ' },
    })
    fireEvent.change(screen.getByLabelText(text('settings.noHpLabel')), {
      target: { value: '0812-3456-7890' },
    })
    fireEvent.click(screen.getByRole('button', { name: text('common.save') }))

    await vi.waitFor(() => expect(mocks.updateProfile).toHaveBeenCalledTimes(1))
    const args = mocks.updateProfile.mock.calls[0][0]
    expect(args.nama).toBe('Budi Santoso')
    expect(args.noHp).toBe('081234567890')
  })
})
