# Runbook insiden SIMPEG

Dokumen ini berisi tindakan yang harus dilakukan operator ketika ada gangguan.
Setiap bagian memuat: gejala, diagnosis, tindakan, dan hal yang **tidak** boleh
dilakukan. Jalankan bagian yang relevan, catat kronologi, dan hubungi pemilik
proses bila langkah perlu keputusan di luar wewenang teknis.

Aturan umum:

- **Jangan pernah menghapus atau mengubah baris `auditLog`.** Jejak audit bersifat
  append-only. Penghapusan hanya mungkin lewat prosedur arsip yang disetujui
  pemilik kebijakan, dan prosedur itu belum ditetapkan.
- **Jangan mengirim data pegawai ke luar sistem** saat diagnosis. Gunakan ID, kode
  error, dan korelasi request ID (`X-Request-ID`), bukan isi data.
- **Jangan mengubah skema database di production** tanpa migration yang sudah
  direview dan diuji di lingkungan terpisah.

## 1. Server tidak mau start

**Gejala:** proses berhenti saat start dengan pesan `config: ...` atau
`database connection setup failed`.

**Diagnosis:**

```bash
# Periksa variabel wajib
echo "${APP_ENV}" "${DATABASE_URL:+db-url-diset}" "${BETTER_AUTH_SECRET:+secret-diset}" "${TRUSTED_ORIGINS:+origins-diset}"
```

**Tindakan:**

1. Pastikan `DATABASE_URL` disetel dan menunjuk ke instance yang benar.
2. Pada `APP_ENV=production`, `BETTER_AUTH_SECRET` wajib ada, minimal 32 karakter,
   dan tidak boleh memuat pola bawaan seperti `change-me`.
3. Pada `APP_ENV=production`, `TRUSTED_ORIGINS` wajib diisi dan tidak boleh `*`.
4. Pastikan nilai pool database masuk akal: `DB_MAX_IDLE_CONNS` tidak boleh lebih
   besar dari `DB_MAX_OPEN_CONNS`.

**Jangan:** inserting nilai default untuk production agar start berhasil.

## 2. Database tidak dapat dijangkau

**Gejala:** `GET /ready` mengembalikan `503` dengan `checks.postgres.status` =
`unavailable`, atau log akses mencatat banyak `503`.

**Diagnosis:**

```bash
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/ready
```

Readiness menjalankan ping database dengan batas waktu (default 15 detik).
`GET /health` tetap `200` selama proses hidup — itu liveness, bukan kesiapan.

**Tindakan:**

1. Periksa apakah instance database aktif dan tidak sedang undergoing maintenance.
2. Periksa batas koneksi: `DB_MAX_OPEN_CONNS`mungkin perlu diturunkan bila
   instance dibagi dengan layanan lain dan mengalami connection exhaustion.
3. Jika hanya sebagian endpoint yang lambat, periksa `DB_STATEMENT_TIMEOUT`.
   Query yang melewati batas dibatalkan oleh driver PostgreSQL.

**Jangan:** menaikkan `DB_STATEMENT_TIMEOUT` tanpa investigating query yang
lambat — batas waktu ada untuk mencegah sumber daya habis.

## 3. Antrean atau timeout pada permintaan

**Gejala:** permintaan sering `503`, atau durasi pada log akses jauh melewati
batas normal.

**Tindakan:**

1. Lihat `duration_ms` pada log akses untuk mengidentifikasi endpoint lambat.
2. Periksa `totalItems` pada halaman daftar — endpoint daftar memakai pagination
   server-side dengan `pageSize` maksimum 100.
3. Pastikan tidak ada laporan yang meminta `pageSize` besar; nilai di luar batas
   ditolak dengan `400 invalid_query`.

## 4. Kebocoran kredensial

**Gejala:** ada dugaan password atau token bocor ke log, respons API, atau repo.

**Tindakan (urutan):**

1. **Putuskan akses lebih dulu.** Jika kredensial aktif, rotasi segera di
   secret manager. Rotasi lebih cepat daripada investigasi.
2. Periksa log akses. Header `Authorization`, body request, dan nilai query
   string tidak pernah dicatat oleh `internal/observability` — bila kredensial
   terlihat di sana, berarti ada jalur pencatatan lain yang perlu diperiksa.
3. Periksa respons API: bentuk error seragam hanya memuat `code`, `message`,
   `requestId`, `fields`, dan `rows`, tidak memuat nilai internal.
4. Periksa repository: `git log -p -S '<fragmen>'` untuk mencari apakah kredensial
   pernah masuk komitman. Riwayat commit tidak dapat "dihapus" dengan aman —
   seluruh kredensial yang pernah masuk komitman harus dianggap bocor.
5. Catat insiden beserta request ID terkait dan tindakan yang dilakukan.

**Jangan:** menampilkan isi `.env`, nilai `DATABASE_URL`, atau isi token di
issue, pesan, atau dokumen.

## 5. Akun atau sesi harus dicabut segera

**Gejala:**Pegawai keluar dari sistem, ada kecurigaan penyalahgunaan, atau
persona yang dinonaktifkan masih aktif.

**Tindakan:**

1. Backend menolak sesi untuk akun nonaktif secara otomatis: akun dengan
   `aktif = false` ditolak sebagai sesi tidak valid (`401`).
2. Cabut sesi dengan `POST /api/v1/auth/logout`. Endpoint ini menghapus baris
   `session` sehingga token tidak lagi berlaku, dan mencatat jejak `session.revoked`.
3. Untuk pencabutan massal, hapus baris `session` milik pengguna terkait
   **melalui prosedur operasional yang disetujui**, bukan dengan operasional
   manual terhadap database di luar aplikasi.

**Jangan:** mengedit baris `user.roleId` atau `user.aktif` langsung di database.
Perubahan role harus melalui endpoint yang berjejak audit.

## 6. Gangguan integrasi eksternal (Redis, QStash, SMTP, S3)

Integrasi tersebut belum diimplementasikan. Bila integrasi sudah aktif:

1.Setiap integrasi wajib punya fallback ke sumber kebenaran di PostgreSQL.
   Redis dan queue **tidak boleh** menjadi sumber kebenaran untuk data pegawai,
   payroll, atau pembayaran.
2. Callback QStash diverifikasi signature-nya terhadap raw request body sebelum
   diproses, dan handler harus idempoten karena retry dapat mengulang delivery.
3. Notifikasi gaji hanya dikirim setelah konfirmasi pembayaran tersimpan sebagai
   terkonfirmasi, bukan saat slip diterbitkan.

## 7. Pemulihan setelah penerapan migrasi

**Sebelum deploy:**

1. Tinjau seluruh migrasi yang belum diterapkan, termasuk urutan dan kompatibilitas
   mundur. Migrasi di `backend/migrations/` memakai nomor berurutan.
2. Jalankan di lingkungan terpisah dengan data sintetis lebih dulu.
3. Pastikan backup dan restore sudah diuji, bukan hanya backup.

**Setelah deploy:**

1. Verifikasi `GET /ready` mengembalikan `200`.
2. Verifikasi `GET /health` tetap `200`.
3. Cek log akses untuk error dan request ID bermasalah.

**Jika migrasi gagal di tengah jalan:**

1. Jangan menjalankan migrasi berikutnya secara manual.
2. Tentukan apakah migrasi bersifat reversible. Migrasi yang menambah kolom
   (`000003_audit_log_unit.sql`) dapat dibalik dengan `DROP COLUMN`; migrasi yang
   menambah kolom dengan `NOT NULL` tanpa default memerlukan langkah pemulihan
   data lebih dulu.
3. Dokumentasikan tindakan pada catatan deployment.

## 8. Skema dan mapping GORM tidak sinkron

Gejala: query gagal dengan `column "..." does not exist`, atau tes paritas skema
gagal.

**Diagnosis:** periksa status dan urutan Goose migrations, lalu jalankan
`go test ./internal/database/...` untuk memeriksa keberadaan tabel model,
indeks jalur query kritis, dan cakupan migrasi RLS. Tes ini tidak membuktikan
bahwa migrasi sudah diterapkan atau seluruh kolom GORM cocok dengan database.

**Tindakan:** verifikasi skema database target terhadap migrasi Goose dan
perbaiki migrasi/model melalui perubahan berversi yang ditinjau. Jangan memakai
`AutoMigrate` untuk memperbaiki skema production.
