import { expect, test, type Page } from '@playwright/test'

/**
 * Pengujian halaman manajemen pegawai memakai data sintetis. Setiap
 * panggilan API disimulasikan lewat page.route supaya pengujian tidak
 * pernah menyentuh data pegawai nyata.
 */

const ACCOUNT = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'operator.uji@instansi.go.id',
  roleName: 'operator',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: ['employee:read:homebase', 'organization:read:homebase'],
}

const UNIT_ID = '22222222-2222-4222-8222-222222222222'

const ORGANIZATION_PAYLOAD = {
  items: [
    {
      id: UNIT_ID,
      name: 'Bagian Kepegawaian',
      description: null,
      jenis: 'BAGIAN',
      parentId: null,
      aktif: true,
      createdAt: '2026-01-01T00:00:00.000Z',
      updatedAt: '2026-01-01T00:00:00.000Z',
    },
  ],
  page: 1,
  pageSize: 100,
  totalItems: 1,
  totalPages: 1,
}

const EMPLOYEE_PAYLOAD = {
  items: [
    {
      id: '11111111-1111-4111-8111-111111111111',
      nama: 'Siti Rahma',
      email: 'siti.rahma@instansi.go.id',
      nip: '198001012010012001',
      nidn: null,
      nidk: null,
      kategoriPegawaiId: null,
      homebaseId: UNIT_ID,
      jabatanStruktural: 'Kepala Bagian',
      jabatanFungsional: null,
      aktif: true,
      createdAt: '2026-01-01T00:00:00.000Z',
      updatedAt: '2026-01-01T00:00:00.000Z',
    },
  ],
  page: 1,
  pageSize: 20,
  totalItems: 1,
  totalPages: 1,
}

async function stubSession(page: Page) {
  await page.route('**/api/v1/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(ACCOUNT),
    })
  })

  await page.route('**/api/auth/sign-in/email', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        token: 'token-e2e-sintetis',
        user: { id: ACCOUNT.userId, email: ACCOUNT.email, name: 'Operator Uji' },
      }),
    })
  })

  await page.route('**/api/v1/organizations*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(ORGANIZATION_PAYLOAD),
    })
  })
}

async function stubEmployees(page: Page, payload: unknown, status = 200) {
  await page.route('**/api/v1/employees*', async (route) => {
    await route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(payload),
    })
  })
}

async function signInAndOpenEmployees(page: Page) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(ACCOUNT.email)
  await page.getByLabel('Kata sandi').fill('kata-sandi-e2e')
  await page.getByRole('button', { name: 'Masuk', exact: true }).click()
  await expect(page).toHaveURL(/\/workspace$/)

  // Pada layar lebar (desktop) tautan "Pegawai" tampil di header; pada ponsel
  // tersembunyi dalam drawer sidebar yang harus dibuka terlebih dahulu.
  const pegawai = page.getByRole('link', { name: 'Pegawai' }).first()
  if (!(await pegawai.isVisible())) {
    await page.getByRole('button', { name: 'Buka menu navigasi' }).click()
  }
  await pegawai.click()
  await expect(page).toHaveURL(/\/employees$/)
}

test.describe('halaman manajemen pegawai', () => {
  test('daftar pegawai tampil beserta nama unitnya', async ({ page }) => {
    await stubSession(page)
    await stubEmployees(page, EMPLOYEE_PAYLOAD)

    await signInAndOpenEmployees(page)

    await expect(page.getByRole('heading', { name: 'Manajemen Pegawai' })).toBeVisible()
    await expect(page.getByText('Siti Rahma')).toBeVisible()
    await expect(page.getByText('Bagian Kepegawaian')).toBeVisible()
    await expect(page.getByText('Kepala Bagian')).toBeVisible()
  })

  test('radar loader tampil selama permintaan daftar tertunda', async ({ page }) => {
    await stubSession(page)

    let release: (() => void) | undefined
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })

    await page.route('**/api/v1/employees*', async (route) => {
      await gate
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(EMPLOYEE_PAYLOAD),
      })
    })

    await signInAndOpenEmployees(page)

    await expect(page.getByRole('status')).toBeVisible()
    await expect(page.getByText('Memuat data pegawai')).toBeVisible()

    release?.()
    await expect(page.getByText('Siti Rahma')).toBeVisible()
  })

  test('permintaan yang gagal menampilkan keadaan galat dan tombol coba lagi', async ({
    page,
  }) => {
    await stubSession(page)
    await stubEmployees(page, { error: { code: 'internal_error' } }, 500)

    await signInAndOpenEmployees(page)

    await expect(page.getByText('Data pegawai tidak dapat dimuat')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Coba lagi' })).toBeVisible()
  })

  test('tombol coba lagi mengulang permintaan dan memuat daftar', async ({ page }) => {
    await stubSession(page)

    let attempt = 0
    await page.route('**/api/v1/employees*', async (route) => {
      attempt += 1
      await route.fulfill({
        status: attempt === 1 ? 500 : 200,
        contentType: 'application/json',
        body:
          attempt === 1
            ? JSON.stringify({ error: { code: 'internal_error' } })
            : JSON.stringify(EMPLOYEE_PAYLOAD),
      })
    })

    await signInAndOpenEmployees(page)

    await expect(page.getByText('Data pegawai tidak dapat dimuat')).toBeVisible()
    await page.getByRole('button', { name: 'Coba lagi' }).click()

    await expect(page.getByText('Siti Rahma')).toBeVisible()
    expect(attempt).toBe(2)
  })

  test('mengurutkan kolom nama meneruskan parameter ke server', async ({ page }) => {
    await stubSession(page)

    const requests: string[] = []
    await page.route('**/api/v1/employees*', async (route) => {
      requests.push(route.request().url())
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(EMPLOYEE_PAYLOAD),
      })
    })

    await signInAndOpenEmployees(page)
    await expect(page.getByText('Siti Rahma')).toBeVisible()

    await page.getByRole('button', { name: /^Nama/ }).click()
    await expect.poll(() => requests.at(-1)).toContain('sort=nama')
  })

  test('filter unit meneruskan homebaseId ke server', async ({ page }) => {
    await stubSession(page)

    const requests: string[] = []
    await page.route('**/api/v1/employees*', async (route) => {
      requests.push(route.request().url())
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(EMPLOYEE_PAYLOAD),
      })
    })

    await signInAndOpenEmployees(page)
    await expect(page.getByText('Siti Rahma')).toBeVisible()

    await page.getByRole('combobox', { name: 'Unit' }).click()
    await page.getByRole('option', { name: 'Bagian Kepegawaian' }).click()

    await expect.poll(() => requests.at(-1)).toContain('homebaseId=')
  })

  test('halaman tidak meluap mendatar pada lebar 375px', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 })
    await stubSession(page)
    await stubEmployees(page, EMPLOYEE_PAYLOAD)

    await signInAndOpenEmployees(page)
    await expect(page.getByText('Siti Rahma')).toBeVisible()

    const overflow = await page.evaluate(() => {
      const root = document.documentElement
      return root.scrollWidth - root.clientWidth
    })

    expect(overflow).toBeLessThanOrEqual(0)
  })

  test('cakupan tanpa pegawai menampilkan keadaan kosong', async ({ page }) => {
    await stubSession(page)
    await stubEmployees(page, {
      ...EMPLOYEE_PAYLOAD,
      items: [],
      totalItems: 0,
      totalPages: 0,
    })

    await signInAndOpenEmployees(page)

    await expect(page.getByText('Belum ada pegawai')).toBeVisible()
    await expect(page.getByRole('table')).toHaveCount(0)
  })
})
