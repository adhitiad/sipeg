# Checklist Staging & Produksi SIMPEG

> Untuk dipakai saat lingkungan staging/produksi representatif tersedia. Dokumen ini **bukan** bukti bahwa rilis sudah terjadi; ia mendaftarkan pemeriksaan yang harus dituntaskan — dengan bukti — sebelum dan sesudah rilis. Item di sini yang belum dapat diverifikasi harus tetap terbuka.

Rujukan: `docs/RUNBOOK.md`, `docs/IMPLEMENTATION-PLAN.md` (§6, §7, §9), `docs/DECISIONS.md`, `backend/README.md`, `docs/SECURITY-TEST-OWASP-2025.md`, `docs/UI-LOAD-TEST.md`.

## 0. Prasyarat lingkungan

- [ ] Environment GitHub (`staging`, `production`) dengan secret terpisah: `DATABASE_URL`, `BETTER_AUTH_SECRET`, `PII_ENCRYPTION_KEY`, `TRUSTED_ORIGINS`, `APP_BASE_URL`, kredensial SMTP/QStash/Redis (bila dipakai), `CLOUDINARY_*` (bila dipakai).
- [ ] `APP_ENV=production` mem-validasi secret produksi (menolak start bila `BETTER_AUTH_SECRET`/`PII_ENCRYPTION_KEY` kosong/tidak valid; `TRUSTED_ORIGINS` wajib).
- [ ] DNS, TLS (sertifikat valid, HSTS), dan domain aplikasi disepakati; origin browser terdaftar di `TRUSTED_ORIGINS`.
- [ ] Backup & PITR diaktifkan dan **di-restore-drill** pada database target (bukan hanya disposable lokal, lihat `DEC-050`); salinan `PII_ENCRYPTION_KEY` tersimpan di vault; rotasi/backfill direncanakan bila kunci diganti.
- [ ] Migrasi diterapkan lewat workflow `deploy-migrations.yml` (atau `cmd/migrate`) pada **staging lebih dulu**, diverifikasi (`migrate status`), kemudian produksi dengan persetujuan pemilik; `000024_jabatan.sql` belum diterapkan di produksi per catatan keputusan — terapkan bila disetujui. Catatan: `000025`–`000033` (penggajian inti, presensi, cuti, kategori, slip `BATAL`, RBAC scoped, indeks `rbac_user_role_scopes`) terpasang pada database target uji/staging per 16 November 2026 (goose versi **33**, `perfbench schema` selaras) — produksi menunggu persetujuan pemilik.
- [ ] Seed RBAC scoped (`backend/cmd/rbacseed`) dijalankan sesuai `docs/RUNBOOK.md` §7A: dry-run → verifikasi `perfbench schema` → `--apply` di staging → uji pencabutan (403) → baru produksi dengan persetujuan pemilik.

## 1. Keamanan dan konfigurasi produksi

- [ ] Header keamanan di lapisan deployment (bukan hanya dev/preview): CSP, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy`, HSTS; CSP `connect-src` hanya origin yang diperlukan.
- [ ] CORS hanya untuk origin di `TRUSTED_ORIGINS`; wildcard tidak dipakai; kredensial lintas origin diputuskan sadar.
- [ ] Rate limit, rai batas body, dan `DB_STATEMENT_TIMEOUT` diverifikasi pada konfigurasi produksi; batas pool < batas pooler Supabase.
- [ ] Rotasi semua kredensial yang pernah dibagikan/terekspos; verifikasi tidak ada rahasia di repo/bundle klien; Dependabot/govulncheck terpantau.
- [ ] Ulangi pemindaian OWASP (ZAP full/active) terhadap staging; triage temuan terbuka; risiko residual diterima tertulis oleh pemilik (termasuk F-09 CSP `unsafe-inline`).
- [ ] RLS default-deny diverifikasi pada database target (bukan hanya `simpeg_perf`/uji lokal); policy RLS yang menambah cakupan hanya dengan persetujuan.
- [ ] `BETTER_AUTH_SECRET` produksi terpisah dari semua environment lain; sesi Better Auth dan resolusi GORM backend selaras (DEC-049); pencabutan dicoba nyata.

## 2. Fungsional dan data

> Bukti disposable (2026-11-16, **bukan** staging): alur gaji diverifikasi pada DB disposable `simpeg_gaji_e2e` (migrasi `000001`–`000033` bersih) dengan data sintetis — generate (`SL-2026-12-0001`, bruto Rp 4.250.000) → `terbit` → `payment` (TRX-UJI-001) → **TERBAYAR**; notifikasi `email.penggajian` ter-antre di `outboxMessage` **tanpa nominal**; verifikasi publik slip tampil nama/periode/status tanpa nominal; jejak audit `payroll.slip.generated`/`payroll.slip.published`/`payroll.payment.confirmed` terisi. Bukti lengkap di `docs/IMPLEMENTATION-PLAN.md` (item operasional) dan `docs/DECISIONS.md` (`DEC-098`).

- [ ] Smoke test pascadeploy: `/health` (200), `/ready` (200/503 sesuai dependency), masuk/keluar, `GET /api/v1/me` (daftar `permissions` terisi — bila kosong, `PolicyAdapter`/RBAC belum tersambung), `GET /api/v1/employees` (cakupan sesuai peran), slip gaji, logout memutus sesi.
- [ ] Modul master pensiun: `GET /api/v1/pensiun/konfigurasi` & `GET /api/v1/pensiun/kepesertaan` sebagai admin (200) dan sebagai operator (200 **terbatas unit operator**), `POST` sebagai operator (403), pegawai (403).
- [ ] Modul baca Surat Keputusan: `GET /api/v1/surat-keputusan` sebagai admin (200, seluruh SK) dan sebagai operator (200 **hanya SK cakupan unit**), detail SK di luar cakupan → `404`, pegawai → `403`.
- [ ] Migrasi skema `user`/`session`/`Role`/PII selaras dengan model (`perfbench schema` terhadap database target); `000023_keluarga_nik_encrypt.sql` tervalidasi tanpa baris legacy plaintext.
- [ ] Provisioning akun pilot bernama (admin/operator) dengan unit penugasan; kata sandi sekali pakai tidak dicatat; MFA untuk role berisiko jika diputuskan; Google OAuth verifikasi live bila `GOOGLE_CLIENT_*` diisi.
- [ ] Notifikasi gaji: alur "konfirmasi pembayaran (tabel `pembayaran`) → antrean → email tanpa nominal" dicoba nyata ke mailbox uji; idempotensi diperiksa (retry tidak mengirim ganda).
- [ ] Kalender libur (`HOLIDAY_API_*`) opsional; kosong = dinonaktifkan dengan tertib.

## 3. Performa dan kapasitas

- [ ] Baseline ulang `perfbench` pada topologi target; pool koneksi, `pgbouncer`/pooler diuji; tidak ada `EMAXCONNSESSION` pada beban puncak yang disepakati.
- [ ] Uji beban UI di lingkungan representatif (bangunan lambat/jaringan terbatas): indikator utama (masuk, daftar pegawai halaman 100, pencarian) terukur dan terdokumentasi (lihat `docs/UI-LOAD-TEST.md` untuk metode); LCP/INP diperiksa (Lighthouse/Web Vitals).
- [ ] Uji beban backend pada endpoint tersibuk (daftar pegawai, slip gaji, ekspor audit) dengan `perfbench e2e`; hasil dimasukkan `docs/PERFORMANCE-BASELINE.md` dengan batasnya.
- [ ] Impor/ekspor massal diuji dengan data sintetis 5.000–50.000 baris; impor tidak melebihi batas request/statement timeout.

## 4. Observabilitas dan operasi

- [ ] `SLO_ENABLED` diputuskan; dashboard `/admin/slo` kosong ≠ tidak ada trafik — bandingkan dengan jumlah permintaan nyata; alert dipasang pada infrastruktur (bukan hanya ambang runbook); pemilik menerima notifikasi uji.
- [ ] Log JSON terkirim ke agregator; tidak memuat kredensial/body/nilai query; `X-Request-ID` konsisten end-to-end; retensi log produksi ditetapkan.
- [ ] `AUDIT_RETENTION_DAYS` diputuskan; `audit.Cleaner` terjadwal via QStash/RabbitMQ sesuai mode; ekspor audit dicoba dan tercatat sebagai `audit.exported`.
- [ ] Runbook diuji untuk insiden minimal: readiness gagal, DB tidak dapat diakses, SMTP mati (antrean outbox tumbuh → dead-letter), Redis mati (fallback PostgreSQL), penjadwal berhenti.
- [ ] Rollback aplikasi (image/frontend sebelumnya) dan rollback migrasi (atau rencana kompatibilitas) diuji di staging.

## 5. Rilis

- [ ] Gate CI hijau (backend & frontend) di commit yang akan dirilis; `gofmt`, `staticcheck`, `govulncheck`, pemindaian kredensial.
- [ ] Persetujuan pemilik produk + keamanan + operasi tertulis; catatan keputusan di `docs/DECISIONS.md`; tanggal dan penanggung jawab rilis.
- [ ] Canary/rollout bertahap atau minimal maintenance window; verifikasi pascadeploy; rencana rollback dengan kriteria jelas.
- [ ] Komunikasi ke pilot: apa yang berubah, cara melaporkan masalah, kanal dukungan.

## 6. Pascarilis

- [ ] Pantau 24–72 jam pertama: error rate per rute, latensi p95, health dependency, antrean outbox (pending/failed), kegagalan notifikasi, 5xx.
- [ ] Backup/restore drill periodik (awal: bulanan); tinjauan akses berkala; peninjauan temuan OWASP; tindak lanjut keputusan terbuka di `docs/DECISIONS.md` (mis. kebijakan MFA wajib, nama akun pilot, verifikasi Google OAuth nyata).
