import { expect, test, type Page } from '@playwright/test'

/**
 * Pengujian halaman admin pemetaan PIN mesin absensi & presensi. Data API
 * disimulasikan; yang diperiksa adalah perilaku antarmuka.
 */

const ADMIN = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'admin.gaji@instansi.go.id',
  roleName: 'admin',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: ['attendance:presensi:manage', 'attendance:read:any'],
}

const PINS = [
  {
    id: '00000000-0000-4000-8000-0000000000e1',
    machinePin: '1001',
    userId: 'a0000000-0000-4000-8000-000000000002',
    nama: 'Pegawai Sintetis',
    aktif: true,
    catatan: 'Mesin Finger 06',
    createdAt: '14-11-2026 10.00 WIB',
  },
]

async function stubAdmin(page: Page) {
  await page.route('**/api/v1/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(ADMIN),
    })
  })
  await page.route('**/api/auth/sign-in/email', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        token: 'token-e2e-sintetis',
        user: { id: ADMIN.userId, email: ADMIN.email, name: 'Admin Uji' },
      }),
    })
  })
  await page.route('**/api/v1/attendance/mesin-pin', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: PINS }),
      })
      return
    }
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify(PINS[0]),
    })
  })
  await page.route('**/api/v1/attendance/mesin-pin/*', async (route) => {
    await route.fulfill({ status: 204, body: '' })
  })
}

async function signIn(page: Page) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(ADMIN.email)
  await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page).toHaveURL(/\/workspace$/)
}

test('admin melihat halaman mesin absensi dan pemetaan PIN', async ({ page }) => {
  await stubAdmin(page)
  await signIn(page)

  await page.getByRole('link', { name: 'Mesin Absensi' }).click()
  await expect(
    page.getByRole('heading', { name: 'Mesin Absensi & Presensi' }),
  ).toBeVisible()
  await expect(page.getByRole('cell', { name: '1001', exact: true })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'Pegawai Sintetis' })).toBeVisible()
})

test('admin dapat membuka dialog tambah pemetaan PIN', async ({ page }) => {
  await stubAdmin(page)
  await signIn(page)

  await page.getByRole('link', { name: 'Mesin Absensi' }).click()
  await expect(
    page.getByRole('heading', { name: 'Mesin Absensi & Presensi' }),
  ).toBeVisible()
  await page.getByRole('button', { name: 'Tambah pemetaan' }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(page.getByLabel('PIN mesin')).toBeVisible()
  await expect(page.getByLabel('Pegawai (userId)')).toBeVisible()
})
