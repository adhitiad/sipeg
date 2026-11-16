// Tetapkan role RBAC ke akun pilot.
// Jalankan setelah roleId statis terisi (pilot-roleid) dan rbacseed --apply
// pada DB target. Idempoten: assignment yang sudah ada tidak digandakan.
// Usage: bun run scripts/assign-rbac.ts
import pg from 'pg'

const ASSIGNMENTS = [
  { email: 'adhitiadwima@gmail.com', roleCode: 'SUPERADMIN' }, // tier 1
  { email: 'uewfau@gmail.com', roleCode: 'OPERATOR_INDUK' }, // tier 2
]

async function main() {
  const connectionString = process.env.DATABASE_URL
  if (!connectionString) throw new Error('DATABASE_URL wajib diisi')
  const pool = new pg.Pool({
    connectionString,
    max: 5,
    // Supabase pooler tidak menampilkan rantai CA publik yang dikenal
    // Node; untuk alat operator ini, verifikasi sertifikat dimatikan
    // (koneksi tetap terenkripsi TLS).
    ssl: { rejectUnauthorized: false },
  })
  try {
    for (const a of ASSIGNMENTS) {
      const { rows: users } = await pool.query(`SELECT id FROM "user" WHERE email = $1`, [
        a.email,
      ])
      if (users.length === 0) {
        console.warn(`User tidak ditemukan: ${a.email}`)
        continue
      }
      const userId = users[0].id
      const { rows: roles } = await pool.query(
        `SELECT id FROM rbac_roles WHERE code = $1`,
        [a.roleCode],
      )
      if (roles.length === 0) {
        console.warn(`Role tidak ditemukan: ${a.roleCode}`)
        continue
      }
      const roleId = roles[0].id
      const { rows: existing } = await pool.query(
        `SELECT 1 FROM rbac_user_role_scopes WHERE user_id = $1 AND role_id = $2 LIMIT 1`,
        [userId, roleId],
      )
      if (existing.length > 0) {
        console.log(`Tetap: ${a.roleCode} -> ${a.email} sudah ter-assign`)
        continue
      }
      await pool.query(
        `INSERT INTO rbac_user_role_scopes (user_id, role_id, created_at, updated_at)
         VALUES ($1,$2,now(),now())`,
        [userId, roleId],
      )
      console.log(
        `Assigned ${a.roleCode} -> ${a.email} (user_id=${userId}, role_id=${roleId})`,
      )
    }
    console.log(
      'Selesai. Verifikasi: SELECT ur.user_id, r.code FROM rbac_user_role_scopes ur JOIN rbac_roles r ON r.id = ur.role_id;',
    )
  } finally {
    await pool.end()
  }
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
