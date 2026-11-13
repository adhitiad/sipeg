import { expect, test, type Page } from '@playwright/test'

const UNAUTHORIZED_BODY = {
  error: { code: 'unauthorized', message: 'Sesi tidak valid atau sudah berakhir.' },
}

const SYNTHETIC_ACCOUNT = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'pegawai.uji@instansi.go.id',
  roleName: 'operator',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: ['employee:read:homebase', 'employee:manage:homebase'],
}

const ADMIN_ACCOUNT = {
  ...SYNTHETIC_ACCOUNT,
  email: 'admin.uji@instansi.go.id',
  roleName: 'admin',
  permissions: ['employee:read:any', 'slo:read', 'audit:read'],
}

/** Data sintetis: backend API disimulasikan agar pengujian tidak memakai data pegawai nyata. */
async function stubApi(
  page: Page,
  options: { signedIn: boolean; account?: typeof SYNTHETIC_ACCOUNT },
) {
  const account = options.account ?? SYNTHETIC_ACCOUNT

  await page.route('**/api/v1/me', async (route) => {
    if (!options.signedIn) {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify(UNAUTHORIZED_BODY),
      })
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(account),
    })
  })

  await page.route('**/api/auth/sign-in/email', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        token: 'token-e2e-sintetis',
        user: {
          id: account.userId,
          email: account.email,
          name: 'Pegawai Uji',
        },
      }),
    })
  })

  await page.route('**/api/auth/sign-out', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
  })
}

const SLO_METRICS_PAYLOAD = {
  windowStart: '2026-11-13T00:00:00.000+07:00',
  windowEnd: '2026-11-14T00:00:00.000+07:00',
  requests: 1280,
  errors: 12,
  errorRate: 0.0094,
  avgMs: 41.2,
  p95Ms: 820,
  p99Ms: 1300,
  maxMs: 2100,
  routes: [
    {
      route: '/api/v1/employees',
      method: 'GET',
      requests: 1280,
      errors: 12,
      errorRate: 0.0094,
      avgMs: 41.2,
      p95Ms: 820,
      p99Ms: 1300,
      maxMs: 2100,
    },
  ],
}

const SLO_METRICS_EMPTY = {
  ...SLO_METRICS_PAYLOAD,
  requests: 0,
  errors: 0,
  errorRate: 0,
  avgMs: 0,
  p95Ms: 0,
  p99Ms: 0,
  maxMs: 0,
  routes: [],
}

// Metrik SLO disimulasikan; pengujian tidak memakai metrik dari proses nyata.
async function stubSloMetrics(page: Page, payload: unknown, status = 200) {
  await page.route('**/api/v1/admin/slo/metrics*', async (route) => {
    await route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(payload),
    })
  })
}

async function signIn(page: Page, email: string) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page).toHaveURL(/\/workspace$/)
}

test.describe('beranda publik', () => {
  test('menampilkan halaman depan dan tautan masuk', async ({ page }) => {
    await stubApi(page, { signedIn: false })

    await page.goto('/')

    await expect(
      page.getByRole('heading', { name: 'Data kepegawaian dalam satu tempat' }),
    ).toBeVisible()
    await expect(page.getByRole('link', { name: 'Masuk' }).first()).toBeVisible()
  })

  test('tautan masuk pada halaman depan menuju halaman masuk', async ({ page }) => {
    await stubApi(page, { signedIn: false })

    await page.goto('/')
    await page.getByRole('link', { name: 'Masuk' }).first().click()

    await expect(page).toHaveURL(/\/login$/)
    await expect(page.getByRole('heading', { name: 'Masuk ke SIMPEG' })).toBeVisible()
  })

  test('halaman hukum dapat dibuka tanpa sesi', async ({ page }) => {
    await stubApi(page, { signedIn: false })

    for (const [path, heading] of [
      ['/about', 'Tentang SIMPEG'],
      ['/privacy', 'Kebijakan Privasi'],
      ['/terms', 'Ketentuan Layanan'],
      ['/terms-of-reference', 'Terminologi Acuan'],
    ] as const) {
      await page.goto(path)
      await expect(page.getByRole('heading', { name: heading })).toBeVisible()
    }
  })

  test('alamat yang tidak dikenal menampilkan not-found', async ({ page }) => {
    await stubApi(page, { signedIn: false })

    await page.goto('/halaman-yang-tidak-ada')

    await expect(
      page.getByRole('heading', { name: 'Halaman tidak ditemukan' }),
    ).toBeVisible()
  })
})

test.describe('halaman masuk', () => {
  test('validasi formulir memakai bahasa Indonesia', async ({ page }) => {
    await stubApi(page, { signedIn: false })

    await page.goto('/login')
    await page.getByRole('button', { name: 'Masuk', exact: true }).click()

    await expect(page.getByText('Format email tidak valid')).toBeVisible()
    await expect(page.getByText('Kata sandi wajib diisi')).toBeVisible()
  })

  test('tamu diarahkan ke halaman masuk saat membuka ruang kerja', async ({ page }) => {
    await stubApi(page, { signedIn: false })

    await page.goto('/workspace')

    await expect(page).toHaveURL(/\/login$/)
  })

  test('masuk membuka ruang kerja dan menampilkan identitas', async ({ page }) => {
    await stubApi(page, { signedIn: true })

    await page.goto('/login')
    await page.getByLabel('Email').fill(SYNTHETIC_ACCOUNT.email)
    await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
    await page.getByRole('button', { name: 'Masuk', exact: true }).click()

    await expect(page).toHaveURL(/\/workspace$/)
    await expect(
      page.getByRole('heading', {
        name: `Anda masuk sebagai ${SYNTHETIC_ACCOUNT.email}`,
      }),
    ).toBeVisible()
    await expect(
      page.getByRole('definition').filter({ hasText: /^Operator$/ }),
    ).toBeVisible()
    await expect(page.getByText('employee:read:homebase')).toBeVisible()
  })

  test('avatar dan menu akun tampil setelah masuk', async ({ page }) => {
    await stubApi(page, { signedIn: true })

    await page.goto('/login')
    await page.getByLabel('Email').fill(SYNTHETIC_ACCOUNT.email)
    await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
    await page.getByRole('button', { name: 'Masuk', exact: true }).click()

    await expect(page).toHaveURL(/\/workspace$/)

    await page.getByRole('button', { name: 'Akun saya' }).click()
    await expect(page.getByRole('menuitem', { name: 'Ruang kerja' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Keluar' })).toBeVisible()
  })

  test('navigasi utama menuju halaman yang tersedia', async ({ page }) => {
    await stubApi(page, { signedIn: true })

    // Token hanya hidup di memori tab; ruang kerja hanya dapat dibuka setelah
    // masuk lewat antarmuka, bukan dengan goto langsung.
    await page.goto('/login')
    await page.getByLabel('Email').fill(SYNTHETIC_ACCOUNT.email)
    await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
    await page.getByRole('button', { name: 'Masuk', exact: true }).click()

    await expect(page).toHaveURL(/\/workspace$/)
    await page.getByRole('link', { name: 'Tentang' }).click()

    await expect(page).toHaveURL(/\/about$/)
    await expect(page.getByRole('heading', { name: 'Tentang SIMPEG' })).toBeVisible()
  })
})

test.describe('bahasa dan tema', () => {
  test('peralihan bahasa mengubah label navigasi', async ({ page }) => {
    await stubApi(page, { signedIn: false })

    await page.goto('/')

    // Klik dapat terjadi sebelum hidrasi selesai; ulangi sampai teks berubah.
    await expect(async () => {
      await page.getByRole('button', { name: 'English' }).click()
      await expect(
        page.getByRole('heading', { name: 'Staff data in one place' }),
      ).toBeVisible({ timeout: 1000 })
    }).toPass()

    await expect(async () => {
      await page.getByRole('button', { name: 'Bahasa Indonesia' }).click()
      await expect(
        page.getByRole('heading', { name: 'Data kepegawaian dalam satu tempat' }),
      ).toBeVisible({ timeout: 1000 })
    }).toPass()
  })

  test('tombol tema menukar antara terang dan gelap', async ({ page }) => {
    await stubApi(page, { signedIn: false })

    await page.goto('/')
    const html = page.locator('html')

    await expect(html).toHaveAttribute('data-theme', 'light')

    await expect(async () => {
      await page.getByRole('button', { name: 'Beralih ke tema gelap' }).click()
      await expect(html).toHaveAttribute('data-theme', 'dark', { timeout: 1000 })
    }).toPass()

    await expect(async () => {
      await page.getByRole('button', { name: 'Beralih ke tema terang' }).click()
      await expect(html).toHaveAttribute('data-theme', 'light', { timeout: 1000 })
    }).toPass()
  })

  test('tidak ada luapan horizontal pada layar ponsel', async ({ page }) => {
    await stubApi(page, { signedIn: false })

    await page.setViewportSize({ width: 375, height: 720 })
    await page.goto('/')

    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )

    expect(overflow).toBeLessThanOrEqual(0)
  })
})

test.describe('keyboard', () => {
  test('urutan Tab pada halaman masuk mencakup tema, wordmark, dan formulir', async ({
    page,
  }) => {
    await stubApi(page, { signedIn: false })

    await page.goto('/login')
    await expect(page.getByRole('button', { name: 'Masuk', exact: true })).toBeVisible()

    await page.keyboard.press('Tab')
    await expect(
      page.getByRole('button', { name: 'Beralih ke tema gelap' }),
    ).toBeFocused()

    await page.keyboard.press('Tab')
    await expect(page.getByRole('link', { name: 'SIMPEG' })).toBeFocused()

    await page.keyboard.press('Tab')
    await expect(page.getByLabel('Email')).toBeFocused()

    await page.keyboard.press('Tab')
    await expect(page.getByLabel('Kata sandi')).toBeFocused()

    await page.keyboard.press('Tab')
    await expect(page.getByRole('button', { name: 'Masuk', exact: true })).toBeFocused()
  })

  test('menu akun dapat dibuka dan ditutup dengan keyboard', async ({ page }) => {
    await stubApi(page, { signedIn: true })

    await page.goto('/login')
    await page.getByLabel('Email').fill(SYNTHETIC_ACCOUNT.email)
    await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
    await page.getByRole('button', { name: 'Masuk', exact: true }).click()

    await expect(page).toHaveURL(/\/workspace$/)

    const accountButton = page.getByRole('button', { name: 'Akun saya' })
    await accountButton.focus()
    await page.keyboard.press('Enter')

    await expect(page.getByRole('menuitem', { name: 'Ruang kerja' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Keluar' })).toBeVisible()

    await page.keyboard.press('Escape')

    await expect(page.getByRole('button', { name: 'Keluar' })).toBeHidden()
    await expect(accountButton).toBeFocused()
  })
})

test.describe('metrik SLO admin', () => {
  test('admin melihat tautan Metrik SLO dan tabel metrik', async ({ page }) => {
    await stubApi(page, { signedIn: true, account: ADMIN_ACCOUNT })
    await stubSloMetrics(page, SLO_METRICS_PAYLOAD)

    await signIn(page, ADMIN_ACCOUNT.email)

    const navLink = page.getByRole('link', { name: 'Metrik SLO' })
    await expect(navLink).toBeVisible()
    await navLink.click()

    await expect(page).toHaveURL(/\/admin\/slo$/)
    await expect(page.getByRole('heading', { name: 'Metrik SLO' })).toBeVisible()
    await expect(page.getByRole('rowheader', { name: '/api/v1/employees' })).toBeVisible()
    await expect(page.getByRole('cell', { name: '820 ms' })).toBeVisible()
  })

  test('operator tidak melihat tautan Metrik SLO', async ({ page }) => {
    await stubApi(page, { signedIn: true })
    await stubSloMetrics(page, SLO_METRICS_PAYLOAD)

    await signIn(page, SYNTHETIC_ACCOUNT.email)

    await expect(page.getByRole('link', { name: 'Metrik SLO' })).toHaveCount(0)
  })

  test('admin melihat keadaan kosong saat belum ada metrik terkumpul', async ({
    page,
  }) => {
    await stubApi(page, { signedIn: true, account: ADMIN_ACCOUNT })
    await stubSloMetrics(page, SLO_METRICS_EMPTY)

    await signIn(page, ADMIN_ACCOUNT.email)
    await page.getByRole('link', { name: 'Metrik SLO' }).click()

    await expect(page.getByText('Belum ada metrik terkumpul')).toBeVisible()
    await expect(page.getByRole('table')).toHaveCount(0)
  })

  test('keadaan galat tampil saat endpoint metrik gagal', async ({ page }) => {
    await stubApi(page, { signedIn: true, account: ADMIN_ACCOUNT })
    await stubSloMetrics(page, { error: { code: 'service_unavailable' } }, 503)

    await signIn(page, ADMIN_ACCOUNT.email)
    await page.getByRole('link', { name: 'Metrik SLO' }).click()

    await expect(page.getByText('Metrik tidak dapat dimuat')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Coba lagi' })).toBeVisible()
  })
})
