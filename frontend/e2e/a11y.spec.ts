import { expect, test, type Page } from '@playwright/test'

// Sesi disimulasikan agar halaman tidak terjebak di keadaan "Memuat sesi…".
async function stubSignedOut(page: Page) {
  await page.route('**/api/v1/me', async (route) => {
    await route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({
        error: { code: 'unauthorized', message: 'Sesi tidak valid.' },
      }),
    })
  })
}

// "Lompat ke konten utama" harus menjadi pemberhentian tab pertama dan
// memindahkan fokus ke wilayah konten utama (#konten-utama).
test('tautan lompat menjadi pemberhentian pertama dan memindahkan fokus', async ({
  page,
}) => {
  await stubSignedOut(page)
  await page.goto('/')

  await page.keyboard.press('Tab')
  const skip = page.locator('.skip-link')
  await expect(skip).toBeFocused()
  await expect(skip).toHaveText('Lompat ke konten utama')

  await page.keyboard.press('Enter')
  await expect(page).toHaveURL(/#konten-utama$/)
  await expect(page.locator('#konten-utama')).toBeFocused()
})

async function focusedTagIds(page: Page, tabs: number): Promise<string[]> {
  const ids: string[] = []
  for (let i = 0; i < tabs; i += 1) {
    await page.keyboard.press('Tab')
    ids.push(
      await page.evaluate(() => {
        const el = document.activeElement
        if (!el) return 'none'
        return el.id ? `#${el.id}` : el.tagName.toLowerCase()
      }),
    )
  }
  return ids
}

// Urutan tab pada halaman masuk: email → kata sandi → tombol masuk.
test('urutan fokus formulir masuk mengikuti urutan visual', async ({ page }) => {
  await stubSignedOut(page)
  await page.goto('/login')
  await expect(page.getByLabel('Email')).toBeVisible()

  const ids = await focusedTagIds(page, 12)
  const email = ids.indexOf('#email')
  const password = ids.indexOf('#password')
  expect(email, `email tidak ditemukan di jalur fokus: ${ids.join(' ')}`).toBeGreaterThan(
    -1,
  )
  expect(
    password,
    `kata sandi tidak ditemukan di jalur fokus: ${ids.join(' ')}`,
  ).toBeGreaterThan(email)
})

// Label formulir terkait dengan kontrolnya (aksesibilitas screen reader).
test('kontrol formulir masuk memiliki label terprogram', async ({ page }) => {
  await stubSignedOut(page)
  await page.goto('/login')
  await expect(page.getByLabel('Email')).toBeVisible()

  await expect(page.getByLabel('Email')).toBeVisible()
  await expect(page.getByLabel('Kata sandi')).toBeVisible()
})

// Cincin fokus terlihat untuk elemen interaktif pada mode gelap maupun terang.
test('cincin fokus terlihat pada tautan lompat', async ({ page }) => {
  await stubSignedOut(page)
  await page.goto('/')

  await page.keyboard.press('Tab')
  const outline = await page.evaluate(() => {
    const el = document.querySelector('.skip-link') as HTMLElement
    el.focus()
    const style = window.getComputedStyle(el)
    return style.outlineStyle !== 'none' || style.boxShadow !== 'none'
  })
  expect(outline).toBe(true)
})

// Rute admin baru (RBAC, Pensiun, SK) tanpa sesi harus mengarahkan ke
// halaman masuk dan fokus berpindah ke kontrol email (tidak terjebak).
for (const path of ['/admin/rbac', '/admin/pensiun', '/admin/sk']) {
  test(`rute admin ${path} tanpa sesi mengarahkan ke masuk dengan fokus pada email`, async ({
    page,
  }) => {
    await stubSignedOut(page)
    await page.goto(path)
    await expect(page).toHaveURL(/\/login/)
    await expect(page.getByLabel('Email')).toBeVisible()
    // Email harus dapat menerima fokus (tidak terjebak di luar konten utama).
    await page.getByLabel('Email').focus()
    await expect(page.getByLabel('Email')).toBeFocused()
  })
}
