import { expect, test, type Page } from '@playwright/test'

/**
 * Alur masuk dengan autentikasi dua langkah (TOTP) memakai data sintetis.
 * Semua panggilan API disimulasikan; pengujian tidak menyentuh autentikasi nyata.
 */

const ACCOUNT = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'operator.uji@instansi.go.id',
  roleName: 'operator',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: ['employee:read:homebase', 'organization:read:homebase'],
}

async function stubAccount(page: Page) {
  await page.route('**/api/v1/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(ACCOUNT),
    })
  })
}

async function signInViaMfa(page: Page) {
  await page.route('**/api/auth/sign-in/email', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ twoFactorRedirect: true, twoFactorMethods: ['totp'] }),
    })
  })
  await page.route('**/api/auth/two-factor/verify-totp', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        token: 'token-e2e-2fa',
        user: { id: ACCOUNT.userId, email: ACCOUNT.email, name: 'Operator Uji' },
      }),
    })
  })
  await page.goto('/login')
  await page.getByLabel('Email').fill(ACCOUNT.email)
  await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page).toHaveURL(/\/mfa$/)
}

test.describe('autentikasi dua langkah', () => {
  test('tantangan 2FA dapat diverifikasi dan membuka ruang kerja', async ({ page }) => {
    await stubAccount(page)
    await signInViaMfa(page)

    await expect(
      page.getByRole('heading', { name: 'Autentikasi Dua Langkah' }),
    ).toBeVisible()
    await page.getByLabel('Kode verifikasi').fill('123456')
    await page.getByRole('button', { name: 'Verifikasi' }).click()
    await expect(page).toHaveURL(/\/workspace$/)
  })

  test('kode cadangan dapat dipakai sebagai pengganti TOTP', async ({ page }) => {
    await stubAccount(page)
    await page.route('**/api/auth/two-factor/verify-backup-code', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          token: 'token-e2e-backup',
          user: { id: ACCOUNT.userId, email: ACCOUNT.email, name: 'Operator Uji' },
        }),
      })
    })
    await signInViaMfa(page)

    await expect(
      page.getByRole('heading', { name: 'Autentikasi Dua Langkah' }),
    ).toBeVisible()
    await page.getByRole('button', { name: 'Pakai kode cadangan' }).click()
    await page.getByLabel('Kode verifikasi').fill('abcde-12345')
    await page.getByRole('button', { name: 'Verifikasi' }).click()
    await expect(page).toHaveURL(/\/workspace$/)
  })
})
