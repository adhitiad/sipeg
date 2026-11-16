import { expect, test } from '@playwright/test'

/**
 * Uji beban UI (manual, tidak berjalan di CI). Hanya aktif bila PERF_STACK=1
 * dan stack penuh (backend :8099 + DB disposable hasil perfbench seed
 * --employees=5000) berjalan di http://localhost:3103.
 *
 * Contoh: PERF_STACK=1 bun --bun run e2e -- perf.spec.ts
 */
const ENABLED = process.env.PERF_STACK === '1'
const BASE = process.env.PERF_BASE_URL ?? 'http://localhost:3103'
const EMAIL = process.env.PERF_EMAIL ?? 'loadadmin@simpeg-perf.test'
const PASSWORD = process.env.PERF_PASSWORD ?? ''

test.skip(!ENABLED, 'Uji beban UI hanya berjalan bila PERF_STACK=1')

test('uji beban UI: halaman pegawai dengan 5000 data', async ({ page }) => {
  const timings: Record<string, number> = {}
  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(String(error)))

  // 1. Muat halaman masuk dan ukur navigation timing.
  await page.goto(`${BASE}/login`, { waitUntil: 'load' })
  timings.loginLoadMs = await page.evaluate(() => {
    const [nav] = performance.getEntriesByType(
      'navigation',
    ) as PerformanceNavigationTiming[]
    return Math.round(nav.loadEventEnd)
  })

  // 2. Masuk; ukur dari klik sampai ruang kerja tampil.
  await expect(page.getByLabel('Email')).toBeVisible()
  const tSignIn = Date.now()
  await page.getByLabel('Email').fill(EMAIL)
  await page.getByLabel('Kata sandi').fill(PASSWORD)
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page).toHaveURL(/\/workspace$/)
  timings.signInMs = Date.now() - tSignIn

  // 3. Navigasi ke halaman pegawai; ukur klik → tabel terisi (20 baris).
  const tEmployees = Date.now()
  await page.getByRole('link', { name: 'Pegawai' }).click()
  await expect(page.locator('tbody tr')).toHaveCount(20)
  timings.employeesNavMs = Date.now() - tEmployees

  // 4. Ubah ukuran halaman ke 100; ukur sampai 100 baris tampil.
  const tSize = Date.now()
  await page.getByLabel('Tampilkan').click()
  await page.getByRole('option', { name: '100' }).click()
  await expect(page.locator('tbody tr')).toHaveCount(100)
  timings.pageSize100Ms = Date.now() - tSize

  // 5. Pencarian yang tidak cocok; ukur sampai keadaan kosong tampil.
  const tEmpty = Date.now()
  await page.getByLabel('Cari pegawai').fill('zzz-tidak-ada')
  await expect(
    page.getByText('Tidak ada pegawai yang cocok dengan pencarian atau cakupan Anda.'),
  ).toBeVisible()
  timings.searchNoMatchMs = Date.now() - tEmpty

  // 5b. Keadaan kosong menyediakan tombol untuk mengatur ulang pencarian.
  const tReset = Date.now()
  await page.getByRole('button', { name: 'Atur ulang pencarian' }).click()
  await expect(page.locator('tbody tr').first()).toBeVisible()
  timings.resetFiltersMs = Date.now() - tReset

  // 6. Pencarian sempit; ukur sampai baris pertama tampil.
  const tSearch = Date.now()
  await page.getByLabel('Cari pegawai').fill('Pegawai Sintetis 000123')
  await expect(page.locator('tbody tr').first()).toBeVisible()
  timings.searchNarrowMs = Date.now() - tSearch

  console.log(`PERF_TIMINGS ${JSON.stringify(timings)}`)
  console.log(`PERF_PAGE_ERRORS ${JSON.stringify(errors)}`)
  expect(errors).toEqual([])
})
