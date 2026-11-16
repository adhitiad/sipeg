import { expect, test, type Page } from '@playwright/test'

/**
 * Pengujian UI slip gaji: daftar, detail, tombol Unduh PDF, dan dialog
 * Generate slip. Data API disimulasikan.
 */

const ADMIN = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'admin.gaji@instansi.go.id',
  roleName: 'admin',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: ['payroll:manage', 'payroll:read:any'],
}

const SLIP = {
  id: '00000000-0000-4000-8000-00000000f001',
  nomorSlip: 'SL-2026-06-0001',
  userId: 'a0000000-0000-4000-8000-000000000002',
  nama: 'Pegawai Sintetis',
  periode: '2026-06',
  periodeLabel: 'Juni 2026',
  status: 'TERBIT',
  statusLabel: 'Terbit',
  tanggalGaji: '01-06-2026',
  gajiPokok: 'Rp 9.500.000',
  pendapatan: [{ jenis: 'FUNGSIONAL', nilai: 'Rp 1.500.000' }],
  potongan: [{ jenis: 'ALPA', nominal: 'Rp 75.000' }],
  totalGaji: 'Rp 11.000.000',
  tanggalDibayar: null,
}

const SLIP_DRAF = {
  ...SLIP,
  id: '00000000-0000-4000-8000-00000000f002',
  status: 'DRAF',
  statusLabel: 'Draf',
}

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
  await page.route(/\/api\/v1\/payroll\/slips\?/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [SLIP],
        page: 1,
        pageSize: 10,
        totalItems: 1,
        totalPages: 1,
      }),
    })
  })
  await page.route(`**/api/v1/payroll/slips/${SLIP.id}`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(SLIP),
    })
  })
  await page.route('**/api/v1/payroll/slips/generate', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(SLIP),
    })
  })
  await page.route(`**/api/v1/payroll/slips/${SLIP_DRAF.id}/terbit`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ...SLIP_DRAF, status: 'TERBIT', statusLabel: 'Terbit' }),
    })
  })
}

async function signIn(page: Page) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(ADMIN.email)
  await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page).toHaveURL(/\/workspace$/)
}

test('admin melihat daftar slip dan membuka dialog generate', async ({ page }) => {
  await stubAdmin(page)
  await signIn(page)

  await page.getByRole('link', { name: 'Slip gaji' }).click()
  await expect(page.getByRole('heading', { name: /Slip Gaji/i })).toBeVisible()

  await page.getByRole('button', { name: 'Generate slip' }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(page.getByLabel('Pegawai (userId)')).toBeVisible()
  await expect(page.getByLabel('Periode')).toBeVisible()
})

test('admin membuka detail slip dan melihat tombol unduh PDF', async ({ page }) => {
  await stubAdmin(page)
  await signIn(page)

  await page.getByRole('link', { name: 'Slip gaji' }).click()
  await page.getByRole('link', { name: 'Lihat detail' }).click()
  await expect(page.getByRole('button', { name: 'Unduh PDF' })).toBeVisible()
})

test('admin menerbitkan slip DRAF menjadi TERBIT', async ({ page }) => {
  await stubAdmin(page)
  await page.route(`**/api/v1/payroll/slips/${SLIP_DRAF.id}`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(SLIP_DRAF),
    })
  })
  await page.route(/\/api\/v1\/payroll\/slips\?/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [SLIP_DRAF],
        page: 1,
        pageSize: 10,
        totalItems: 1,
        totalPages: 1,
      }),
    })
  })
  await signIn(page)

  await page.getByRole('link', { name: 'Slip gaji' }).click()
  await page.getByRole('link', { name: 'Lihat detail' }).click()
  await expect(page.getByRole('button', { name: 'Terbitkan', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Terbitkan', exact: true }).click()
  await expect(page.getByText('Slip gaji diterbitkan.')).toBeVisible()
})

test('admin melihat slip yang memuat komponen THR memiliki badge THR di daftar', async ({
  page,
}) => {
  await stubAdmin(page)
  await page.route(/\/api\/v1\/payroll\/slips\?/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [
          {
            ...SLIP_DRAF,
            id: '00000000-0000-4000-8000-00000000f003',
            nomorSlip: 'THR-2027-0001',
            pendapatan: [{ jenis: 'THR', nilai: 'Rp 11.000.000' }],
          },
        ],
        page: 1,
        pageSize: 10,
        totalItems: 1,
        totalPages: 1,
      }),
    })
  })
  await signIn(page)

  await page.getByRole('link', { name: 'Slip gaji' }).click()
  await expect(page.getByText('THR-2027-0001')).toBeVisible()
  await expect(page.getByText('THR', { exact: true }).first()).toBeVisible()
})
