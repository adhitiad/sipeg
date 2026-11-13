import { createElement } from 'react'
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LocaleProvider } from '@/i18n/locale'
import LoadingBars from '@/components/LoadingBars'
import LoadingRadar from '@/components/LoadingRadar'

function harness(child: React.ReactElement) {
  return createElement(LocaleProvider, null, child)
}

describe('indikator memuat', () => {
  it('batang menandai status dan membacakan labelnya', () => {
    render(harness(createElement(LoadingBars, { label: 'Memuat halaman' })))

    const status = screen.getByRole('status')
    expect(status).toHaveAttribute('aria-busy', 'true')
    expect(status).toHaveAttribute('aria-label', 'Memuat halaman')
  })

  it('radar menandai status dan membacakan labelnya', () => {
    render(harness(createElement(LoadingRadar, { label: 'Memuat daftar pegawai' })))

    const status = screen.getByRole('status')
    expect(status).toHaveAttribute('aria-busy', 'true')
    expect(status).toHaveAttribute('aria-label', 'Memuat daftar pegawai')
  })

  it('mode dekoratif menyembunyikan indikator dari pembaca layar', () => {
    const { container } = render(
      harness(createElement(LoadingBars, { label: 'Memuat', decorative: true })),
    )

    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    expect(container.firstElementChild).toHaveAttribute('aria-hidden', 'true')
  })

  it('radar memakai token warna produk, bukan warna literal', () => {
    const { container } = render(
      harness(createElement(LoadingRadar, { label: 'Memuat' })),
    )

    expect(container.innerHTML).toContain('var(--lagoon)')
    expect(container.innerHTML).toContain('var(--palm)')
    expect(container.innerHTML).not.toContain('rose-500')
    expect(container.innerHTML).not.toContain('stone-200')
  })

  it('batang memakai token warna produk, bukan warna literal', () => {
    const { container } = render(harness(createElement(LoadingBars, { label: 'Memuat' })))

    expect(container.innerHTML).toContain('var(--lagoon)')
    expect(container.innerHTML).toContain('var(--palm)')
    expect(container.innerHTML).not.toContain('blue-500')
    expect(container.innerHTML).not.toContain('purple-500')
  })
})
