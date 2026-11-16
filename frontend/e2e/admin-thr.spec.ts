import { expect, test, type Page } from '@playwright/test'

/**
 * Pengujian UI admin untuk halaman THR (konfigurasi, rencana, prorata).
 * Data API disimulasikan.
 */

const ADMIN = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'admin.thr@instansi.go.id',
  roleName: 'admin',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: ['thr:manage'],
}

const CONFIG = {
  tahun: 2027,
  besaranTipe: 'GAJI_KOTOR',
  persen: null,
  syaratMasaKerjaBulan: 3,
  berlakuTanggal: '2027-03-09',
  updatedAt: '2026-11-16T00:00:00Z',
}

const PEGAWAI = {
  userId: 'a0000000-4000-8000-9000-000000000002',
  nama: 'Pegawai Sintetis',
  masaKerjaTahun: 6,
  nominalOtomatis: 11_000_000,
  sumber: 'GAJI_KOTOR',
}

const PLAN = {
  tahun: 2027,
  besaranTipe: 'GAJI_KOTOR',
  berlakuTanggal: '2027-03-09',
  items: [PEGAWAI],
  lewat: [
    {
      userId: 'a0000000-4000-8000-9000-000000000009',
      nama: 'Pegawai Baru',
      reason: 'syarat_masa_kerja',
    },
  ],
}

const REPORT = {
  tahun: 2027,
  besaranTipe: 'GAJI_KOTOR',
  berlakuTanggal: '2027-03-09',
  generated: 1,
  created: [
    {
      userId: PEGAWAI.userId,
      nama: PEGAWAI.nama,
      nominal: 5_500_000,
      sumber: 'input admin (prorata)',
    },
  ],
  skipped: [],
}

async function stubAdmin(page: Page) {
  let generateBody: unknown = null

  await page.route('**/api/auth/sign-in/email', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        token: 'token-e2e-thr',
        user: { id: ADMIN.userId, email: ADMIN.email, name: 'Admin THR' },
      }),
    })
  })
  await page.route('**/api/v1/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(ADMIN),
    })
  })
  await page.route(/\/api\/v1\/payroll\/thr\/config(\?|$)/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [CONFIG] }),
    })
  })
  await page.route('**/api/v1/payroll/thr/config', async (route, request) => {
    if (request.method() === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(CONFIG),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [CONFIG] }),
    })
  })
  await page.route(`**/api/v1/payroll/thr/config/${CONFIG.tahun}`, async (route) => {
    await route.fulfill({ status: 204, body: '' })
  })
  await page.route(/\/api\/v1\/payroll\/thr\/plan/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(PLAN),
    })
  })
  await page.route('**/api/v1/payroll/thr/generate', async (route, request) => {
    generateBody = request.postDataJSON()
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(REPORT),
    })
  })

  return { getGenerateBody: () => generateBody }
}

async function signIn(page: Page) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(ADMIN.email)
  await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page).toHaveURL(/\/workspace$/)
}

test('admin membuka halaman THR, melihat aplikasi, dan meng-generate', async ({
  page,
}) => {
  await stubAdmin(page)
  await signIn(page)

  await page.getByRole('link', { name: 'THR' }).click()
  await expect(page.getByRole('heading', { name: 'THR', exact: true })).toBeVisible()
  await expect(page.getByText('2027', { exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Buat slip THR', exact: true }).click()
  await expect(page.getByText(/Hasil generate THR 2027/i)).toBeVisible()
  await expect(page.getByText(/1 slip dibuat/i)).toBeVisible()
})

test('admin memuat rencana THR dan mengirim nominal prorata', async ({ page }) => {
  const stubs = await stubAdmin(page)
  await signIn(page)

  await page.getByRole('link', { name: 'THR' }).click()
  await expect(page.getByRole('heading', { name: 'THR', exact: true })).toBeVisible()
  await page.getByRole('spinbutton', { name: 'Tahun' }).fill('2027')
  await page.getByRole('button', { name: 'Muat rencana THR', exact: true }).click()

  await expect(page.getByText('Pegawai Sintetis')).toBeVisible()
  await expect(page.getByText('11.000.000')).toBeVisible()

  const input = page.getByLabel(/Nominal dibayar \(prorata\)/)
  await input.fill('5500000')

  await page.getByRole('button', { name: 'Buat slip THR', exact: true }).click()
  await expect(page.getByText(/1 slip dibuat/i)).toBeVisible()

  const body = stubs.getGenerateBody() as {
    tahun: number
    overrides: Array<{ userId: string; nominal: number }>
  }
  expect(body.tahun).toBe(2027)
  expect(body.overrides).toHaveLength(1)
  expect(body.overrides[0]).toEqual({ userId: PEGAWAI.userId, nominal: 5_500_000 })
})
