import { expect, test, type Page } from '@playwright/test'

/**
 * Verifikasi hibrida RBAC: UI sungguhan + backend sungguhan.
 * Hanya autentikasi yang disimulasikan; seluruh panggilan modul RBAC
 * diteruskan ke backend nyata melalui proksi Vite. Dilewati bila backend uji
 * tidak hidup.
 */

const BACKEND = 'http://localhost:8099'
const ADMIN = {
  userId: '9b2b1f4c-6f5f-4f2a-9b1a-5f0f2b6a1c11',
  email: 'admin@example.test',
  roleName: 'admin',
  homebaseId: '4f6f2e5a-9f37-4f5f-8b6b-2a1c9d3e7f21',
  permissions: ['rbac:read', 'rbac:manage'],
}

async function backendHidup(): Promise<boolean> {
  try {
    const resp = await fetch(`${BACKEND}/health`)
    return resp.ok
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

test.describe('RBAC admin (backend nyata)', () => {
  test('admin membuka halaman RBAC dan melihat master role/permission', async ({
    page,
  }) => {
    test.skip(!(await backendHidup()), 'backend uji tidak hidup (localhost:8099)')
    await stubAuth(page)
    await signIn(page)

    await page.getByRole('link', { name: 'RBAC' }).click()
    await expect(page).toHaveURL(/\/admin\/rbac$/)
    await expect(
      page.getByRole('heading', { name: /RBAC|Manajemen RBAC/i }),
    ).toBeVisible()
    await expect(page.getByText(/Role|Peran/i).first()).toBeVisible()
  })
})
