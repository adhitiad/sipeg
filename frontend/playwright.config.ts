import { defineConfig, devices } from '@playwright/test'

// Port khusus E2E. Server dev harian berjalan di port 3000; memakai port
// terpisah memastikan server E2E selalu memakai environment pengujian.
// Kedua alamat API sengaja tidak menunjuk server sungguhan; setiap
// panggilan jaringan disimulasikan lewat page.route pada pengujian.
const APP_PORT = 3101
const APP_URL = `http://localhost:${APP_PORT}`

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  reporter: 'list',
  use: {
    baseURL: APP_URL,
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'bun --bun run dev:e2e',
    url: APP_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    env: {
      VITE_API_BASE_URL: 'http://localhost:8080/api/v1',
    },
  },
})
