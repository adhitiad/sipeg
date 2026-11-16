// Perintah rbacseed menterjemahkan matriks grant statis
// (`internal/policymatrix`, disetujui di docs/PRD.md §11) menjadi data
// RBAC berbasis scoped assignment (`DEC-087`) untuk pengguna yang sudah ada.
//
// Pemakaian:
//
//	rbacseed           # dry-run (bawaan): ringkas apa yang akan dibuat, tanpa menulis
//	rbacseed --apply   # tulis ke database dalam satu transaksi
//
// Lingkungan: DATABASE_URL wajib diisi dan menunjuk database target.
// dry-run tidak mengubah apa pun. --apply bersifat idempoten: peran,
// permission, dan matriks di-upsert; pengguna yang sudah memiliki
// assignment dilewati. Penerapan pada staging/produksi hanya lewat
// prosedur rilis yang disetujui (see docs/STAGING-RELEASE-CHECKLIST.md);
// verifikasi pertama wajib pada database disposable.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"simpeg/backend/internal/policymatrix"
	"simpeg/backend/internal/rbac/domain"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type seedRole struct {
	Code string
	Name string
	Tier int
}

// Peran RBAC yang dibuat selalu lengkap agar dry-run & apply konsisten.
var seedRoles = []seedRole{
	{Code: "SUPERADMIN", Name: "Superadmin", Tier: 1},
	{Code: "OPERATOR_INDUK", Name: "Operator Induk", Tier: 2},
	{Code: "ADMIN", Name: "Admin", Tier: 3},
	{Code: "OPERATOR", Name: "Operator", Tier: 4},
	{Code: "ATASAN", Name: "Atasan", Tier: 4},
	{Code: "PEGAWAI", Name: "Pegawai", Tier: 4},
}

// staticToRBAC menjembatani nama role statis (policy) ke kode peran RBAC.
var staticToRBAC = map[string]string{
	"admin":    "ADMIN",
	"operator": "OPERATOR",
	"atasan":   "ATASAN",
	"pegawai":  "PEGAWAI",
}

func main() {
	args := os.Args[1:]
	apply := false
	for _, arg := range args {
		switch strings.ToLower(strings.TrimSpace(arg)) {
		case "--apply":
			apply = true
		default:
			log.Fatalf("argumen tidak dikenal %q (gunakan --apply untuk menulis; tanpa argumen = dry-run)", arg)
		}
	}
	if err := run(context.Background(), apply); err != nil {
		log.Fatal(err)
	}
}

type summary struct {
	rolesUpserted        int
	permissionsUpserted  int
	rolePermsInserted    int64
	usersAssigned        int64
	usersSkippedAssigned int64
	usersSkippedUnknown  int64
}

func run(ctx context.Context, apply bool) error {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL wajib diisi")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("buka koneksi database: %w", err)
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mulai transaksi: %w", err)
	}
	sum, err := seed(ctx, tx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if apply {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	} else {
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("rollback dry-run: %w", err)
		}
	}

	mode := "DRY-RUN (tidak ada yang ditulis)"
	if apply {
		mode = "APPLY"
	}
	fmt.Printf("rbacseed %s\n", mode)
	fmt.Printf("roles upsert: %d, permissions upsert: %d, matriks baru: %d\n",
		sum.rolesUpserted, sum.permissionsUpserted, sum.rolePermsInserted)
	fmt.Printf("users assignment baru: %d, dilewati (sudah punya assignment): %d, dilewati (role tidak dikenal/null): %d\n",
		sum.usersAssigned, sum.usersSkippedAssigned, sum.usersSkippedUnknown)
	return nil
}

func seed(ctx context.Context, tx *sql.Tx) (summary, error) {
	var sum summary

	// 1. Role master.
	roleIDs := make(map[string]int64, len(seedRoles))
	for _, role := range seedRoles {
		var id int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO rbac_roles (code, name, tier_level)
			VALUES ($1, $2, $3)
			ON CONFLICT (code) DO UPDATE SET code = EXCLUDED.code
			RETURNING id`,
			role.Code, role.Name, role.Tier).Scan(&id)
		if err != nil {
			return sum, fmt.Errorf("upsert role %s: %w", role.Code, err)
		}
		roleIDs[role.Code] = id
		sum.rolesUpserted++
	}

	// 2. Permission master dari gabungan seluruh grant statis.
	matrix := policymatrix.Matrix()
	type permKey struct{ module, action string }
	permIDs := map[permKey]int64{}
	permissionIDByFlat := map[string]int64{}
	for _, grants := range matrix {
		for flat := range grants {
			if _, seen := permissionIDByFlat[flat]; seen {
				continue
			}
			module, action, _ := domain.ParsePermission(flat)
			if module == "" || action == "" {
				return sum, fmt.Errorf("permission tidak dapat dipecah: %q", flat)
			}
			key := permKey{module: module, action: action}
			id, ok := permIDs[key]
			if !ok {
				err := tx.QueryRowContext(ctx, `
					INSERT INTO rbac_permissions (module, action)
					VALUES ($1, $2)
					ON CONFLICT (module, action) DO UPDATE SET module = EXCLUDED.module
					RETURNING id`,
					module, action).Scan(&id)
				if err != nil {
					return sum, fmt.Errorf("upsert permission %s:%s: %w", module, action, err)
				}
				permIDs[key] = id
				sum.permissionsUpserted++
			}
			permissionIDByFlat[flat] = id
		}
	}

	// 3. Matriks role-permission.
	for staticRole, grants := range matrix {
		code, ok := staticToRBAC[staticRole]
		if !ok {
			continue
		}
		roleID := roleIDs[code]
		for flat := range grants {
			permID := permissionIDByFlat[flat]
			res, err := tx.ExecContext(ctx, `
				INSERT INTO rbac_role_permissions (role_id, permission_id)
				VALUES ($1, $2)
				ON CONFLICT (role_id, permission_id) DO NOTHING`,
				roleID, permID)
			if err != nil {
				return sum, fmt.Errorf("matriks %s -> permission %d: %w", code, permID, err)
			}
			affected, err := res.RowsAffected()
			if err != nil {
				return sum, fmt.Errorf("hitung matriks %s: %w", code, err)
			}
			sum.rolePermsInserted += affected
		}
	}

	// 4. Assignment untuk pengguna yang sudah ada.
	rows, err := tx.QueryContext(ctx, `
		SELECT u.id, r.name, u."homebaseId", u."aktif"
		FROM "user" u
		LEFT JOIN "Role" r ON r.id = u."roleId"`)
	if err != nil {
		return sum, fmt.Errorf("baca pengguna: %w", err)
	}
	type userRow struct {
		id       string
		roleName sql.NullString
		homebase sql.NullString
		aktual   bool
	}
	var users []userRow
	for rows.Next() {
		var row userRow
		if err := rows.Scan(&row.id, &row.roleName, &row.homebase, &row.aktual); err != nil {
			rows.Close()
			return sum, fmt.Errorf("baca baris pengguna: %w", err)
		}
		users = append(users, row)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return sum, fmt.Errorf("iterasi pengguna: %w", err)
	}

	for _, row := range users {
		if !row.aktual || !row.roleName.Valid {
			sum.usersSkippedUnknown++
			continue
		}
		code, ok := staticToRBAC[strings.ToLower(strings.TrimSpace(row.roleName.String))]
		if !ok {
			sum.usersSkippedUnknown++
			continue
		}
		var existing int64
		if err := tx.QueryRowContext(ctx,
			`SELECT count(*) FROM rbac_user_role_scopes WHERE user_id = $1`, row.id).Scan(&existing); err != nil {
			return sum, fmt.Errorf("cek assignment pengguna %s: %w", row.id, err)
		}
		if existing > 0 {
			sum.usersSkippedAssigned++
			continue
		}
		// Admin lintas organisasi (unit NULL); operator/atasan/pegawai
		// memakai unit penugasan sebagai informasi cakupan.
		var unitID any
		if code != "ADMIN" && row.homebase.Valid && row.homebase.String != "" {
			unitID = row.homebase.String
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO rbac_user_role_scopes (user_id, role_id, unit_id)
			VALUES ($1, $2, $3)`,
			row.id, roleIDs[code], unitID)
		if err != nil {
			return sum, fmt.Errorf("assignment pengguna %s: %w", row.id, err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return sum, fmt.Errorf("hitung assignment pengguna %s: %w", row.id, err)
		}
		sum.usersAssigned += affected
	}

	return sum, nil
}
