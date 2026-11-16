import { expect, test, type Page } from '@playwright/test'

/**
 * Pengujian UI admin untuk modul baru Tahun 2026-11:
 * pemberhentian (pengajuan, persetujuan, rincian), kontrak (perpanjangan
 * pasca-BUP), dan master aturan pesangon. Data API disimulasikan.
 */

const ADMIN = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'admin.pemberhentian@instansi.go.id',
  roleName: 'admin',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: [
    'pemberhentian:read:any',
    'pemberhentian:propose:any',
    'pemberhentian:approve',
    'kontrak:read:any',
    'perpanjangan:manage',
    'pesangon:manage',
  ],
}

const KARYAWAN_PEMBERHENTIAN = {
  id: '00000000-0000-4000-8000-00000000f101',
  userId: 'a0000000-0000-4000-8000-000000000002',
  jenis: 'BUP',
  alasan: 'BUP',
  tanggal: '2026-11-15',
  status: 'MENUNGGU',
  nomorSk: 'PHK-2026-0001',
  hakTerakhir: null,
  keterangan: null,
  kompensasiBulan: 0,
  createdAt: '2026-11-15T00:00:00Z',
}

const KARYAWAN_PENGAWASAN = {
  ...KARYAWAN_PEMBERHENTIAN,
  id: '00000000-0000-4000-8000-00000000f102',
  status: 'DISETUJUI',
  keterangan: 'pensiun normal',
  kompensasiBulan: 5,
  nomorSk: 'SK-PHK-2026-0001',
}

const KONTRAK_BARU = {
  id: '00000000-0000-4000-8000-00000000c001',
  userId: KARYAWAN_PEMBERHENTIAN.userId,
  jenis: 'PKWT',
  nomorSk: 'PKWT-2026-0001',
  mulai: '2026-11-15',
  selesai: '2027-11-15',
  status: 'AKTIF',
  skId: null,
  createdAt: '2026-11-15T00:00:00Z',
}

const IN_30 = new Date(Date.now() + 9 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
const KONTRAK_SEGERA_BERAKHIR = {
  ...KONTRAK_BARU,
  id: '00000000-0000-4000-8000-00000000c002',
  selesai: IN_30,
  nomorSk: 'PKWT-2026-0000',
}

const ATURAN_PESANGON = {
  id: '00000000-0000-4000-8000-00000000a001',
  kategoriStatusId: 'd0000000-0000-0000-0000-000000000001',
  masaKerjaTahun: 5,
  pesangonBulan: 4,
  penghargaanBulan: 1,
  gantiKerugianBulan: 0,
  sumberRegulasi: 'PP 35/2021',
}

async function stubAdmin(page: Page) {
  await page.route('**/api/auth/sign-in/email', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        token: 'token-e2e-pemberhentian',
        user: { id: ADMIN.userId, email: ADMIN.email, name: 'Admin Uji' },
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

  let proposedBody: unknown = null
  let decidedBody: unknown = null
  let extendBody: unknown = null

  await page.route(/\/api\/v1\/pemberhentian(\?|$)/, async (route) => {
    const url = new URL(route.request().url())
    const status = url.searchParams.get('status') ?? 'MENUNGGU'
    const items = status === 'MENUNGGU' ? [KARYAWAN_PEMBERHENTIAN] : [KARYAWAN_PENGAWASAN]
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items, total: items.length }),
    })
  })
  await page.route('**/api/v1/pemberhentian', async (route, request) => {
    if (request.method() !== 'POST') return route.fallback()
    proposedBody = request.postDataJSON()
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify(KARYAWAN_PEMBERHENTIAN),
    })
  })
  await page.route(
    `**/api/v1/pemberhentian/${KARYAWAN_PEMBERHENTIAN.id}/approve`,
    async (route) => {
      decidedBody = route.request().postDataJSON()
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...KARYAWAN_PEMBERHENTIAN,
          status: 'DISETUJUI',
          kompensasiBulan: 5,
        }),
      })
    },
  )
  await page.route('**/api/v1/pemberhentian/*/approve', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        ...KARYAWAN_PEMBERHENTIAN,
        status: 'DISETUJUI',
        kompensasiBulan: 5,
      }),
    })
  })

  await page.route(/\/api\/v1\/kontrak(\?|$)/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [KONTRAK_SEGERA_BERAKHIR], total: 1 }),
    })
  })
  await page.route('**/api/v1/perpanjangan', async (route, request) => {
    extendBody = request.postDataJSON()
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify(KONTRAK_BARU),
    })
  })

  await page.route('**/api/v1/pesangon-aturan', async (route, request) => {
    if (request.method() === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(ATURAN_PESANGON),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [ATURAN_PESANGON] }),
    })
  })
  await page.route(`**/api/v1/pesangon-aturan/${ATURAN_PESANGON.id}`, async (route) => {
    await route.fulfill({ status: 204, body: '' })
  })

  return {
    getProposedBody: () => proposedBody,
    getDecidedBody: () => decidedBody,
    getExtendBody: () => extendBody,
  }
}

async function signIn(page: Page) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(ADMIN.email)
  await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page).toHaveURL(/\/workspace$/)
}

test('admin membuka halaman pemberhentian, mengajukan, dan menyetujui pengajuan', async ({
  page,
}) => {
  await stubAdmin(page)
  await signIn(page)

  await page.getByRole('link', { name: 'Pemberhentian' }).click()
  await expect(
    page.getByRole('heading', { name: 'Pemberhentian pegawai', exact: true }),
  ).toBeVisible()

  await page.getByRole('button', { name: 'Ajukan pemberhentian', exact: true }).click()
  const propose = page.getByRole('dialog')
  await expect(propose).toBeVisible()
  await propose.getByLabel('UUID pegawai').fill(KARYAWAN_PEMBERHENTIAN.userId)
  await propose.getByRole('combobox', { name: 'Alasan' }).selectOption('BUP')
  await propose.getByLabel('Tanggal efektif').fill('2026-11-15')
  await propose.getByRole('button', { name: 'Simpan', exact: true }).click()
  await expect(page.getByText('Pengajuan pemberhentian dikirim.')).toBeVisible()

  const approve = page.getByRole('button', { name: 'Setujui', exact: true }).first()
  await approve.click()
  const decision = page.getByRole('dialog')
  await expect(decision).toBeVisible()
  await decision.getByRole('button', { name: 'Setujui', exact: true }).click()
  await expect(page.getByText('Pengajuan pemberhentian disetujui.')).toBeVisible()

  await page.getByRole('button', { name: 'Detail', exact: true }).first().click()
  await expect(
    page.getByRole('heading', { name: 'Detail pemberhentian', exact: true }),
  ).toBeVisible()

  const detail = page.getByRole('dialog')
  await expect(detail).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(detail).not.toBeVisible()
})

test('admin melihat kontrak dan mengajukan perpanjangan pasca-BUP', async ({ page }) => {
  await stubAdmin(page)
  await signIn(page)

  await page.getByRole('link', { name: 'Kontrak' }).click()
  await expect(
    page.getByRole('heading', { name: 'Kontrak kerja', exact: true }),
  ).toBeVisible()
  await expect(page.getByText(/Kontrak berakhir dalam 30 hari/)).toBeVisible()

  await page.getByRole('button', { name: 'Perpanjang kontrak', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await dialog.getByLabel('UUID pegawai').fill(KARYAWAN_PEMBERHENTIAN.userId)
  await dialog.getByRole('button', { name: 'Simpan', exact: true }).click()
  await expect(page.getByText('Perjanjian kerja tahunan dibuat.')).toBeVisible()
})

test('admin mengelola aturan pesangon', async ({ page }) => {
  await stubAdmin(page)
  await signIn(page)

  await page.getByRole('link', { name: 'Aturan Pesangon' }).click()
  await expect(
    page.getByRole('heading', { name: 'Aturan pesangon', exact: true }),
  ).toBeVisible()
  await expect(page.getByText(ATURAN_PESANGON.kategoriStatusId)).toBeVisible()

  await page.getByLabel('UUID kategori status').fill(ATURAN_PESANGON.kategoriStatusId)
  await page.getByLabel('Masa kerja (tahun)').fill('0')
  await page.getByLabel('Pesangon (bulan)').fill('2')
  await page.getByLabel('Penghargaan (bulan)').fill('1')
  await page.getByRole('button', { name: 'Simpan', exact: true }).click()

  await page.getByRole('button', { name: 'Hapus', exact: true }).first().click()
})
