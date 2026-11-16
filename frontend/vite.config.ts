import { defineConfig, loadEnv, type Plugin } from 'vite'
import { devtools } from '@tanstack/devtools-vite'

import { tanstackStart } from '@tanstack/react-start/plugin/vite'

import viteReact from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const CSP_POLICY =
  "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"

const SECURITY_HEADERS: Array<readonly [string, string]> = [
  ['Content-Security-Policy', CSP_POLICY],
  ['X-Frame-Options', 'DENY'],
  ['X-Content-Type-Options', 'nosniff'],
  ['Referrer-Policy', 'no-referrer'],
]

const securityHeadersPlugin = (): Plugin => ({
  name: 'simpeg-security-headers',
  configureServer(server) {
    server.middlewares.use((_req, res, next) => {
      for (const [name, value] of SECURITY_HEADERS) {
        res.setHeader(name, value)
      }
      next()
    })
  },
  configurePreviewServer(server) {
    server.middlewares.use((_req, res, next) => {
      for (const [name, value] of SECURITY_HEADERS) {
        res.setHeader(name, value)
      }
      next()
    })
  },
})

const env = loadEnv(process.env.NODE_ENV ?? 'development', process.cwd(), '')
// Target proksi dibaca dari VITE_API_PROXY_TARGET baik dari shell maupun
// dari `.env` (loadEnv); sehingga `bun run dev` polos pun tetap mem-proxy
// /api/v1 ke backend.
const apiProxyTarget = process.env.VITE_API_PROXY_TARGET ?? env.VITE_API_PROXY_TARGET

const config = defineConfig({
  resolve: { tsconfigPaths: true },
  server: apiProxyTarget
    ? {
        proxy: {
          // Proksi same-origin untuk /api/v1 agar pengembangan harian tidak
          // bergantung pada CORS. Target dibaca dari VITE_API_PROXY_TARGET
          // (contoh: http://localhost:8080); bukan alamat yang dibaca antarmuka.
          '/api/v1': {
            target: apiProxyTarget,
            changeOrigin: true,
          },
        },
      }
    : undefined,
  plugins: [
    devtools(),
    tailwindcss(),
    securityHeadersPlugin(),
    tanstackStart({
      router: {
        // Berkas pengujian Vitest diletakkan di src/test. Pola ini menjadi
        // pengaman bila ada berkas .test.* tertinggal di dalam src/routes.
        routeFileIgnorePattern: '\\.test\\.[jt]sx?$',
      },
    }),
    viteReact(),
  ],
})

// routeRules Nitro untuk header keamanan saat build/preview produksi.
// Properti tidak dikenal tipe Vite; runtime Nitro dapat membacanya bila didukung.
;(config as Record<string, unknown>).nitro = {
  routeRules: {
    '/**': {
      headers: Object.fromEntries(SECURITY_HEADERS),
    },
  },
}

export default config
