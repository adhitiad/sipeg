/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Basis URL API backend SIMPEG, contoh: http://localhost:8080/api/v1 */
  readonly VITE_API_BASE_URL?: string
  /** Basis URL runtime autentikasi Better Auth, contoh: http://localhost:8081/api/auth */
  readonly VITE_AUTH_BASE_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
