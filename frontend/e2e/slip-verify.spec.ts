import { expect, test, type Page } from '@playwright/test'

/**
 * Pengujian halaman verifikasi publik slip gaji dan batas unduhan 3× per
 * pegawai per periode (hanya di sisi klien).
 */

const ADMIN = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'admin.gaji@instansi.go.id',
  roleName: 'admin',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: ['payroll:manage', 'payroll:read:any'],
}

const TOKEN = 'abcdefghijklmnopqrstuvwxyz'.slice(0, 22)

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
  pendapatan: [],
  potongan: [],
  totalGaji: 'Rp 11.000.000',
  tanggalDibayar: null,
}

const VERIFY_RESPONSE = {
  nama: 'Pegawai Sintetis',
  periode: '2026-06',
  periodeLabel: 'Juni 2026',
  status: 'TERBIT',
  statusLabel: 'Terbit',
}

async function stubPublic(page: Page) {
  await page.route(`**/api/v1/verify/slip/${TOKEN}`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(VERIFY_RESPONSE),
    })
  })
}

async function stubAdmin(page: Page) {
  let downloads = 0
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
  await page.route(`**/api/v1/payroll/slips/${SLIP.id}/pdf`, async (route) => {
    downloads += 1
    await route.fulfill({
      status: 200,
      contentType: 'application/pdf',
      body: Buffer.from('%PDF-1.4 stub'),
    })
  })
  return () => downloads
}

async function signIn(page: Page) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(ADMIN.email)
  await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page).toHaveURL(/\/workspace$/)
}

test('halaman verifikasi publik menampilkan nama, periode, dan status tanpa nominal', async ({
  page,
}) => {
  await stubPublic(page)
  await page.goto(`/verify/slip/${TOKEN}`)
  await expect(
    page.getByRole('heading', { name: 'Slip gaji terverifikasi' }),
  ).toBeVisible()
  await expect(page.getByText('Pegawai Sintetis')).toBeVisible()
  await expect(page.getByText('Juni 2026')).toBeVisible()
  await expect(page.getByText('Terbit')).toBeVisible()
  await expect(page.getByText('11.000.000')).toHaveCount(0)
})

test('halaman verifikasi dengan token tidak sah menampilkan pesan tidak valid', async ({
  page,
}) => {
  await stubPublic(page)
  await page.goto('/verify/slip/token-bukan-22-karakter-panjang-sekali')
  await expect(
    page.getByRole('heading', { name: 'Verifikasi tidak valid' }),
  ).toBeVisible()
})

test('tombol unduh PDF slip berhenti setelah 3 kali per periode', async ({ page }) => {
  const downloads = await stubAdmin(page)
  await signIn(page)

  await page.getByRole('link', { name: 'Slip gaji' }).click()
  await page.getByRole('link', { name: 'Lihat detail' }).click()
  await expect(page.getByRole('button', { name: 'Unduh PDF' })).toBeVisible()
  await expect(page.getByText('Sisa unduhan: 3')).toBeVisible()

  for (let i = 0; i < 3; i++) {
    await page.getByRole('button', { name: 'Unduh PDF' }).click()
  }
  expect(downloads()).toBe(3)
  await expect(page.getByText('Batas 3 kali unduh per periode tercapai.')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Unduh PDF' })).toBeDisabled()
})
