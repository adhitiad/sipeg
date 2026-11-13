/**
 * Skrip operator untuk mem-provision empat akun uji (admin, operator,
 * atasan, pegawai) pada basis data KOSONG yang baru dimigrasi.
 *
 * HANYA untuk basis data tes/disposable. Jangan jalankan terhadap
 * basis data produksi: skrip ini membuat akun dengan kata sandi acak
 * dan mengekspos kata sandi itu di keluaran terminal.
 *
 * Cara pakai (bash/zsh/PowerShell):
 *   DATABASE_URL=postgres://... BETTER_AUTH_SECRET=... bun scripts/provision-accounts.ts
 *
 * Akan mencetak email dan kata sandi setiap akun ke stdout satu kali.
 */
import { betterAuth } from 'better-auth'
import pg from 'pg'

const ROLE_NAMES = ['admin', 'operator', 'atasan', 'pegawai'] as const

const ROLE_FIXED_IDS: Record<(typeof ROLE_NAMES)[number], string> = {
  admin: '00000000-0000-4000-8000-0000000000a1',
  operator: '00000000-0000-4000-8000-0000000000a2',
  atasan: '00000000-0000-4000-8000-0000000000a3',
  pegawai: '00000000-0000-4000-8000-0000000000a4',
}
const HOMEBASE_FIXED_ID = '00000000-0000-4000-8000-0000000000b1'
const EMAIL_DOMAIN = 'simpeg.local'

function requireEnv(name: string): string {
  const value = process.env[name]?.trim()
  if (!value) {
    throw new Error(`${name} wajib diisi`)
  }
  return value
}

function randomPassword(): string {
  const alphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789'
  const bytes = new Uint8Array(18)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (byte) => alphabet[byte % alphabet.length]).join('')
}

async function main() {
  const connectionString = requireEnv('DATABASE_URL')
  const secret = requireEnv('BETTER_AUTH_SECRET')

  if (!/localhost|127\.0\.0\.1/.test(connectionString)) {
    throw new Error(
      'Skrip ini menolak berjalan terhadap host selain localhost. Pakai basis data lokal disposable.',
    )
  }

  const pool = new pg.Pool({ connectionString, max: 2 })

  // 1. Role dan homebase dengan UUID tetap supaya hasil seeder deterministik.
  for (const name of ROLE_NAMES) {
    await pool.query(
      `INSERT INTO "Role" ("id", "name", "description")
       VALUES ($1, $2, $3)
       ON CONFLICT ("name") DO UPDATE SET "description" = EXCLUDED."description"`,
      [ROLE_FIXED_IDS[name], name, `Role ${name} (akun uji)`],
    )
  }
  await pool.query(
    `INSERT INTO "Homebase" ("id", "name", "description")
     VALUES ($1, 'Unit Uji', 'Unit penugasan untuk akun uji')
     ON CONFLICT ("name") DO UPDATE SET "description" = EXCLUDED."description"`,
    [HOMEBASE_FIXED_ID],
  )

  // 2. Akun dibuat lewat Better Auth (hash scrypt, tabel account terisi).
  const auth = betterAuth({
    baseURL: 'http://localhost:3000',
    secret,
    database: pool,
    advanced: { database: { generateId: () => crypto.randomUUID() } },
    user: { fields: { name: 'nama' } },
    emailAndPassword: { enabled: true, disableSignUp: false, minPasswordLength: 8 },
  })

  const created: { role: string; email: string; password: string }[] = []

  for (const role of ROLE_NAMES) {
    const email = `${role}@${EMAIL_DOMAIN}`
    const password = randomPassword()

    const existing = await pool.query('SELECT "id" FROM "user" WHERE "email" = $1', [
      email,
    ])
    if (existing.rowCount && existing.rowCount > 0) {
      // Idempoten: hapus lalu buat ulang; session/account ikut terhapus lewat ON DELETE CASCADE.
      await pool.query('DELETE FROM "user" WHERE "id" = $1', [
        existing.rows[0].id as string,
      ])
    }
    const result = await auth.api.signUpEmail({
      body: { email, password, name: `Uji ${role}` },
    })
    const userId = result.user.id

    await pool.query(
      'UPDATE "user" SET "roleId" = $1, "homebaseId" = $2, "aktif" = true, "emailVerified" = true WHERE "id" = $3',
      [ROLE_FIXED_IDS[role], HOMEBASE_FIXED_ID, userId],
    )
    created.push({ role, email, password })
  }

  await pool.end()

  console.log()
  console.log('Akun uji dibuat (kata sandi hanya ditampilkan sekali):')
  for (const account of created) {
    console.log(`- ${account.role.padEnd(8)} ${account.email}  ${account.password}`)
  }
}

main().catch((error: unknown) => {
  console.error(error instanceof Error ? error.message : error)
  process.exit(1)
})
