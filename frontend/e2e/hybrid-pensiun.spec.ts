import { expect, test, type Page } from '@playwright/test'

/**
 * Verifikasi hibrida halaman pensiun: UI sungguhan + backend sungguhan bila
 * tersedia. Hanya autentikasi yang disimulasikan; panggilan modul diteruskan
 * ke backend nyata melalui proksi Vite. Dilewati bila backend uji tidak hidup.
 */

const BACKEND = 'http://localhost:8099'
const ADMIN = {
  userId: 'a0000000-4000-8000-9000-000000000001',
  email: 'admin@verify.test',
  roleName: 'admin',
  homebaseId: '20000000-0000-4000-8000-000000000001',
  permissions: ['pensiun:manage'],
  token: 'test-admin-token',
}
const OPERATOR = {
  userId: 'a0000000-4000-8000-9000-000000000002',
  email: 'operator@verify.test',
  roleName: 'operator',
  homebaseId: '20000000-0000-4000-8000-000000000001',
  permissions: ['pensiun:read:homebase'],
  token: 'test-operator-token',
}

async function backendHidup(): Promise<boolean> {
  try {
    const resp = await fetch(`${BACKEND}/health`)
    return resp.ok
  } catch {
    return false
  }
}

async function stubAuth(page: Page, principal: typeof ADMIN) {
  await page.route('**/api/auth/sign-in/email', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        token: principal.token,
        user: {
          id: principal.userId,
          email: principal.email,
          name: 'Prinsipal Sintetis',
        },
      }),
    })
  })
  await page.route('**/api/v1/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(principal),
    })
  })
}

async function signIn(page: Page, principal: typeof ADMIN) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(principal.email)
  await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page).toHaveURL(/\/workspace$/)
}

test.describe('Halaman master Pensiun', () => {
  test('admin melihat formulir konfigurasi dan kepesertaan', async ({ page }) => {
    test.skip(!(await backendHidup()), 'backend uji tidak hidup (localhost:8099)')
    await stubAuth(page, ADMIN)
    await signIn(page, ADMIN)

    await page.getByRole('link', { name: 'Pensiun' }).click()
    await expect(page).toHaveURL(/\/admin\/pensiun$/)
    await expect(
      page.getByRole('heading', { name: /Master Pensiun|Pension Master/i }),
    ).toBeVisible()
    await expect(page.getByRole('button', { name: /Tambah|Add/i }).first()).toBeVisible()
  })

  test('operator read-only: tabel tampil tanpa tombol tambah', async ({ page }) => {
    test.skip(!(await backendHidup()), 'backend uji tidak hidup (localhost:8099)')
    await stubAuth(page, OPERATOR)
    await signIn(page, OPERATOR)

    await page.getByRole('link', { name: 'Pensiun' }).click()
    await expect(page).toHaveURL(/\/admin\/pensiun$/)
    await expect(
      page.getByRole('heading', { name: /Master Pensiun|Pension Master/i }),
    ).toBeVisible()
    await expect(page.getByRole('button', { name: /Tambah|Add/i })).toHaveCount(0)
  })
})
