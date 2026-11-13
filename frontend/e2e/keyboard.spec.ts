import { expect, test, type Page } from '@playwright/test'

/**
 * Pengujian keyboard dan indikator fokus. Data API tetap disimulasikan;
 * hal yang diperiksa di sini adalah perilaku keyboard di antarmuka.
 */

const ACCOUNT = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'operator.uji@instansi.go.id',
  roleName: 'operator',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: ['employee:read:homebase', 'organization:read:homebase'],
}

async function stubSession(page: Page) {
  await page.route('**/api/v1/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(ACCOUNT),
    })
  })
  await page.route('**/api/auth/sign-in/email', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        token: 'token-e2e-sintetis',
        user: { id: ACCOUNT.userId, email: ACCOUNT.email, name: 'Operator Uji' },
      }),
    })
  })
}

async function describeFocused(page: Page): Promise<string> {
  return page
    .evaluate(() => {
      const element = document.activeElement as HTMLElement | null
      if (!element) {
        return ''
      }
      const aria = element.getAttribute('aria-label')
      const id = element.id
      const text = (element.textContent || '').trim().slice(0, 40)
      return aria || id || text || element.tagName.toLowerCase()
    })
    .catch(() => '')
}

async function tabUntil(
  page: Page,
  predicate: (description: string) => boolean,
  maxTabs = 30,
): Promise<string | null> {
  for (let i = 0; i < maxTabs; i++) {
    await page.keyboard.press('Tab')
    const description = await describeFocused(page)
    if (predicate(description)) {
      return description
    }
  }
  return null
}

test.describe('keyboard dan kontras', () => {
  test('halaman masuk dapat dinavigasi dengan keyboard dan fokus terlihat', async ({
    page,
  }) => {
    await page.goto('/login')
    // Tunggu pemeriksaan sesi selesai sehingga formulir masuk tampil.
    await expect(page.getByLabel('Email')).toBeVisible()

    const reached: string[] = []
    for (let i = 0; i < 12; i++) {
      await page.keyboard.press('Tab')
      reached.push(await describeFocused(page))
    }
    const joined = reached.join(' | ')
    expect(joined).toContain('email')
    expect(joined).toContain('password')
    expect(joined).toContain('Masuk')

    // Setiap elemen yang fokus lewat keyboard menunjukkan indikator fokus 2px.
    const focused = page.locator(':focus')
    await expect(focused).toHaveCount(1)
    await expect(focused).toHaveCSS('outline-width', '2px')
    await expect(focused).toHaveCSS('outline-style', 'solid')
  })

  test('urutan Tab setelah masuk mencapai navigasi utama dengan indikator terlihat', async ({
    page,
  }) => {
    await stubSession(page)
    await page.goto('/login')
    await page.getByLabel('Email').fill(ACCOUNT.email)
    await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
    await page.getByRole('button', { name: 'Masuk', exact: true }).click()
    await expect(page).toHaveURL(/\/workspace$/)

    const link = await tabUntil(page, (description) => description === 'Pegawai', 30)
    expect(link).toBe('Pegawai')
    await expect(page.locator(':focus')).toHaveCSS('outline-width', '2px')
  })
})
