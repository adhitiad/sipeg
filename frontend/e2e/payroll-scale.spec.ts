import { expect, test, type Page } from '@playwright/test'

/**
 * Pengujian halaman admin master skala gaji pokok yayasan. Data API
 * disimulasikan; yang diperiksa adalah perilaku antarmuka.
 */

const ADMIN = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'admin.gaji@instansi.go.id',
  roleName: 'admin',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: ['payroll:manage', 'payroll:read:any', 'payroll:read:homebase'],
}

const SKALA = [
  {
    id: '00000000-0000-4000-8000-0000000000a1',
    kode: 'SKJ-2027',
    nama: 'Skala Gaji Yayasan 2027',
    berlakuMulai: '2027-04-01',
    berlakuSelesai: null,
    aktif: true,
    createdAt: '2026-11-14T08:00:00Z',
    updatedAt: '2026-11-14T08:00:00Z',
  },
]

const KATEGORI = [
  {
    id: 'd0000000-0000-4000-8000-000000000001',
    kode: 'DOSEN_TETAP_YAYASAN',
    nama: 'Dosen tetap yayasan',
    deskripsi: 'Dosen tetap yayasan.',
    gajiDariNegara: false,
    butuhKontrak: false,
    wajibBpjs: true,
    bupSesuaiYayasan: true,
    aktif: true,
  },
]

const DETAIL = {
  id: 'd0000000-0000-4000-8000-0000000000d1',
  skalaGajiId: SKALA[0].id,
  kategoriStatusId: KATEGORI[0].id,
  kategoriKode: 'DOSEN_TETAP_YAYASAN',
  kategoriNama: 'Dosen tetap yayasan',
  tingkat: 4,
  nominal: 9750000,
}

async function stubAdmin(page: Page) {
  let created = false
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
  await page.route('**/api/v1/payroll/scale/kategori-status', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(KATEGORI),
    })
  })
  await page.route('**/api/v1/payroll/scale?**', async (route) => {
    const items = created ? [...SKALA] : [...SKALA]
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items,
        page: 1,
        pageSize: 25,
        totalItems: items.length,
        totalPages: 1,
      }),
    })
  })
  await page.route(`**/api/v1/payroll/scale/${SKALA[0].id}`, async (route) => {
    if (route.request().method() === 'POST') {
      created = true
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(SKALA[0]),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ...SKALA[0], details: [DETAIL] }),
    })
  })
  await page.route(`**/api/v1/payroll/scale/${SKALA[0].id}/detail`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([DETAIL]),
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

test('admin melihat halaman skala gaji dan membuka rincian', async ({ page }) => {
  await stubAdmin(page)
  await signIn(page)

  // Token hanya ada di memori; gunakan navigasi dalam aplikasi.
  await page.getByRole('link', { name: 'Skala gaji' }).click()
  await expect(page.getByRole('heading', { name: 'Skala Gaji Pokok' })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'SKJ-2027' })).toBeVisible()

  await page.getByRole('button', { name: 'Rincian' }).first().click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(page.getByRole('cell', { name: 'Dosen tetap yayasan' })).toBeVisible()
  await expect(page.getByRole('cell', { name: /9\.750\.000|9,750,000/ })).toBeVisible()
})

test('admin dapat membuka dialog tambah skala gaji', async ({ page }) => {
  await stubAdmin(page)
  await signIn(page)

  await page.getByRole('link', { name: 'Skala gaji' }).click()
  await expect(page.getByRole('heading', { name: 'Skala Gaji Pokok' })).toBeVisible()
  await page.getByRole('button', { name: 'Tambah skala' }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(page.getByLabel('Kode')).toBeVisible()
  await expect(page.getByLabel('Nama')).toBeVisible()
})
