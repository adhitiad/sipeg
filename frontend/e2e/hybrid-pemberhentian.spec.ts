import { expect, test, type Page } from '@playwright/test'
import { writeFileSync } from 'node:fs'

/**
 * Verifikasi hibrida: UI sungguhan + backend sungguhan + database disposable.
 *
 * Hanya autentikasi yang disimulasikan (sign-in dan `/api/v1/me`); seluruh
 * panggilan modul penggajian diteruskan ke backend nyata melalui proksi Vite,
 * sehingga alur pemberhentian → approve → generate slip diuji sampai ke
 * database. Token sesi yang dipakai adalah token sintetis pada database
 * disposable (`test-admin-token`).
 *
 * Pengujian ini dilewati (skipped) bila backend uji tidak hidup, sehingga
 * `bun run e2e` tanpa backend tetap hijau. Pengujian juga dilewati bila
 * dataset disposable belum siap: alur ini memeriksa nilai nominal_aturan
 * pesangon yang harus sudah diisi (aturan pesangon per kategori status, skala
 * gaji, dan data sakit untuk pegawai uji). Tanpa dataset itu, pratinjau
 * kompensasi menampilkan "aturan belum diisi" — bukan kegagalan produk.
 */

const BACKEND = 'http://localhost:8099'
const ADMIN = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'admin@example.test',
  roleName: 'admin',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: [
    'pemberhentian:read:any',
    'pemberhentian:propose:any',
    'pemberhentian:approve',
    'payroll:manage',
    'payroll:read:any',
    'payroll:schedule',
  ],
}
const PEGAWAI = 'a0000000-4000-8000-9000-000000000002'
const KATEGORI = 'd0000000-0000-0000-0000-000000000001'

async function backendHidup(): Promise<boolean> {
  try {
    const resp = await fetch(`${BACKEND}/health`)
    return resp.ok
  } catch {
    return false
  }
}

/**
 * Memeriksa apakah dataset disposable siap untuk alur penuh: aturan pesangon
 * untuk kategori uji harus sudah terisi (tanpa itu pratinjau kompensasi
 * menampilkan "aturan belum diisi" dan slip tidak memuat PESANGON).
 */
async function datasetSiap(): Promise<boolean> {
  try {
    const resp = await fetch(
      `${BACKEND}/api/v1/pesangon-aturan?kategoriStatusId=${KATEGORI}`,
      { headers: { Authorization: 'Bearer test-admin-token' } },
    )
    if (!resp.ok) return false
    const payload = (await resp.json()) as { items?: unknown[] }
    return Array.isArray(payload.items) && payload.items.length > 0
  } catch {
    return false
  }
}

async function stubAuth(page: Page) {
  await page.route('**/api/auth/sign-in/email', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        token: 'test-admin-token',
        user: { id: ADMIN.userId, email: ADMIN.email, name: 'Admin Sintetis' },
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
}

async function signIn(page: Page) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(ADMIN.email)
  await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page).toHaveURL(/\/workspace$/)
}

test.describe('alur pemberhentian dengan sakit berkepanjangan (backend nyata)', () => {
  test.slow()

  test('approve pemberhentian lalu slip bulan pemberhentian memuat PESANGON', async ({
    page,
  }) => {
    test.skip(!(await backendHidup()), 'backend uji tidak hidup (localhost:8099)')
    test.skip(
      !(await datasetSiap()),
      'dataset disposable belum memiliki aturan pesangon untuk kategori uji',
    )
    await stubAuth(page)
    await signIn(page)

    // 1. Ajukan pemberhentian BUP untuk pegawai (PeriodeSakit bulan 2 = 75%).
    // Navigasi lewat sidebar (bukan page.goto): token sesi disimpan di memori
    // tab, jadi muat ulang halaman akan mengeluarkan pengguna ke /login.
    await page.getByRole('link', { name: 'Pemberhentian' }).click()
    await page.getByRole('button', { name: 'Ajukan pemberhentian', exact: true }).click()
    const propose = page.getByRole('dialog')
    await propose.getByLabel('UUID pegawai').fill(PEGAWAI)
    await propose.getByRole('combobox', { name: 'Alasan' }).selectOption('BUP')
    await propose.getByLabel('Tanggal efektif').fill('2026-11-15')
    await propose.getByRole('button', { name: 'Simpan', exact: true }).click()
    await expect(page.getByText('Pengajuan pemberhentian dikirim.')).toBeVisible()

    // 2. Pratinjau kompensasi sebelum approve (dihitung backend nyata).
    await page.getByRole('button', { name: 'Setujui', exact: true }).first().click()
    const decision = page.getByRole('dialog')
    await expect(decision.getByText('Pratinjau kompensasi')).toBeVisible()
    await expect(decision.getByText('Total bulan gaji pokok')).toBeVisible()

    // 3. Approve → slip bulan pemberhentian memuat PESANGON di atas gaji pokok sakit.
    await decision.getByRole('button', { name: 'Setujui', exact: true }).click()
    await expect(page.getByText('Pengajuan pemberhentian disetujui.')).toBeVisible()

    // 4. Generate slip November 2026 (bulan pemberhentian).
    await page.getByRole('link', { name: 'Slip gaji' }).click()
    await page.getByRole('button', { name: 'Generate slip', exact: true }).click()
    const generate = page.getByRole('dialog')
    await generate.getByLabel('Pegawai (userId)').fill(PEGAWAI)
    await generate.getByLabel('Periode').fill('2026-11')
    await generate.getByLabel('Tingkat skala (opsional)').fill('3')
    await generate.getByLabel('Kategori status pegawai (opsional)').fill(KATEGORI)
    await generate.getByRole('button', { name: 'Generate slip', exact: true }).click()
    await expect(page.getByText(/SL-2026-11-/)).toBeVisible()

    // 5. Detail slip: gaji pokok 75% (sakit) + komponen PESANGON.
    await page.getByRole('link', { name: 'Lihat detail' }).first().click()
    await expect(page.getByText('PESANGON', { exact: true })).toBeVisible()
    await expect(page.getByText('Rp 35.625.000')).toBeVisible()
    await expect(page.getByText('Rp 7.125.000')).toBeVisible()
    await expect(page.getByText('Rp 44.250.000')).toBeVisible()

    // 6. Simpan tangkapan layar dan berkas PDF slip untuk pemeriksaan visual.
    await page.screenshot({
      path: 'C:/Users/humas/AppData/Local/Temp/opencode/slip-pesanganon-detail.png',
      fullPage: true,
    })
    const slipId = page.url().split('/').pop()
    const pdf = await page.request.get(
      `http://localhost:8099/api/v1/payroll/slips/${slipId}/pdf`,
      { headers: { Authorization: 'Bearer test-admin-token' } },
    )
    const bytes = await pdf.body()
    writeFileSync('C:/Users/humas/AppData/Local/Temp/opencode/slip-pesanganon.pdf', bytes)
  })
})
