// Seed akun pilot via Better Auth (hash kata sandi dengan scrypt Better Auth).
// Jalankan: bun run scripts/seed-accounts.ts
// Pastikan DATABASE_URL dan BETTER_AUTH_SECRET ter-set di .env.
import { hashPassword } from '@better-auth/utils/password'
import pg from 'pg'

const ACCOUNTS = [
  {
    email: 'adhitiadwima@gmail.com',
    name: 'Adhitiad Wima',
    password: 'ChangeMe-Admin-2026!',
    role: 'superadmin',
  },
  {
    email: 'uewfau@gmail.com',
    name: 'Operator Induk',
    password: 'ChangeMe-Operator-2026!',
    role: 'operator_induk',
  },
]

async function main() {
  const connectionString = process.env.DATABASE_URL
  if (!connectionString) throw new Error('DATABASE_URL wajib diisi')
  const pool = new pg.Pool({ connectionString, max: 5 })
  try {
    for (const acc of ACCOUNTS) {
      const hash = await hashPassword(acc.password)
      const userId = crypto.randomUUID()
      await pool.query(
        `INSERT INTO "user" (id, nama, email, "emailVerified", aktif, "createdAt", "updatedAt")
         VALUES ($1,$2,$3,true,true,now(),now())
         ON CONFLICT (email) DO NOTHING`,
        [userId, acc.name, acc.email],
      )
      const { rows } = await pool.query(`SELECT id FROM "user" WHERE email = $1`, [
        acc.email,
      ])
      const id = rows[0].id
      await pool.query(
        `INSERT INTO "account" (id, "userId", "accountId", "providerId", password, "createdAt", "updatedAt")
         VALUES ($1,$2,$3,'credential',$4,now(),now())
         ON CONFLICT ("providerId","accountId") DO NOTHING`,
        [crypto.randomUUID(), id, acc.email, hash],
      )
      console.log(
        `Seeded: ${acc.email} (role: ${acc.role}, password: ${acc.password}) — GANTI PASSWORD setelah login pertama`,
      )
    }
    console.log('Selesai. Login via frontend dengan email/kata sandi di atas.')
  } finally {
    await pool.end()
  }
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
