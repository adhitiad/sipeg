# Checklist & formulir uji backup/restore (Supabase Free)

> **Status platform Free:** Supabase Free tidak menyediakan backup database harian otomatis dan tidak menyediakan PITR. Supabase merekomendasikan ekspor data secara teratur menggunakan CLI db dump dan penyimpanan off-site. Ekspor backup adalah pekerjaan operasional yang harus dijadwalkan, dimonitor, dienkripsi, dan diuji — bukan fitur backup otomatis Supabase.
>
> **Hasil drill 2026-11-16 (staging, bukan produksi)** — lengkap di docs/DECISIONS.md DEC-100: dump pg_dump sekitar **59 detik**, 2.555.361 byte; restore ke container disposable postgres:16-alpine sekitar **5 detik** (empat galat spesifik supabase_vault/transaction_timeout dicatat); verifikasi sumber/hasil: 89 tabel, 89 RLS, Goose 33, user=102 (100 sintetis + 2 pilot), gaji=301, session=30, account=2, auditLog=50.161. Artefak dihapus. **Bukan bukti restore pada project Supabase produksi.**
>
> Pemilik: **adhitiad (admin)**. Terkait: docs/DECISIONS.md DEC-050, DEC-065, DEC-100; docs/RUNBOOK.md §8E.
>
> Target awal RPO **24 jam** / RTO **4 jam** adalah sasaran kebijakan, bukan hasil pengukuran produksi. RPO 24 jam pada Free hanya dapat didekati dengan ekspor terjadwal minimal sekali per 24 jam, pemantauan keberhasilan, dan salinan off-site yang valid. RTO harus diukur dari kejadian sampai aplikasi siap, bukan hanya lama menjalankan psql.

## Batas cakupan backup Supabase Free

- Tidak ada backup database harian otomatis pada Free.
- PITR tidak tersedia pada Free; dokumentasi Supabase menyebut PITR sebagai add-on untuk Pro/Team/Enterprise dengan compute Small atau lebih tinggi.
- Supabase CLI db dump default hanya berisi schema. Data dan custom roles harus diekspor dengan --data-only dan --role-only.
- Supabase CLI mengecualikan schema yang dikelola Supabase seperti auth, storage, dan schema extension. Periksa apakah ada objek penting di luar schema aplikasi.
- Backup database tidak mencakup objek file yang disimpan melalui Supabase Storage API. SIMPEG menggunakan Cloudinary untuk dokumen; pemulihan objek Cloudinary harus dibuktikan terpisah dari restore PostgreSQL.
- Restore ke project baru dapat mewarisi default privileges target. Audit akses anon/authenticated, owner, grants, RLS, dan policy sebelum membuka aplikasi.

Referensi resmi:
- https://supabase.com/docs/guides/platform/backups
- https://supabase.com/docs/reference/cli/supabase-issue-feature

## 1) Metadata uji (isi saat eksekusi)

| Field | Nilai |
|---|---|
| Tanggal/waktu mulai dan selesai (WIB/UTC) | |
| Pelaksana | adhitiad (admin) / ... |
| Lingkungan sumber | Production Supabase / staging / disposable |
| Supabase project ref | |
| Supabase tier | Free / ... |
| Region project | |
| Versi Supabase CLI | |
| Versi PostgreSQL project | |
| Commit/tag aplikasi | |
| Timestamp data terakhir sebelum ekspor | |
| Lokasi artefak off-site (tanpa URL rahasia) | |
| Target restore disposable | |
| Metode restore | Supabase project disposable / PostgreSQL disposable |
| Goose version sebelum dan sesudah restore | |

## 2) Verifikasi tier, kemampuan, dan jadwal

- [ ] Dashboard project menunjukkan tier Free dan region aktual dicatat.
- [ ] Tidak ada asumsi backup harian otomatis pada Free.
- [ ] PITR dicatat sebagai tidak tersedia pada tier Free.
- [ ] Jadwal ekspor dibuat minimal sekali setiap 24 jam bila RPO 24 jam masih ditargetkan.
- [ ] Kegagalan job ekspor, checksum, enkripsi, atau pengiriman off-site menghasilkan alert ke operator.
- [ ] Retensi dan lokasi penyimpanan off-site ditetapkan oleh pemilik data.
- [ ] Salinan terenkripsi terpisah dari project Supabase dan host aplikasi.

## 3) Prasyarat & keamanan

- [ ] Supabase CLI versi yang mendukung db dump --remote tersedia; supabase --version dicatat.
- [ ] Docker daemon aktif (CLI menjalankan pg_dump di container).
- [ ] Autentikasi Supabase CLI dan akses project menggunakan akun/credential store yang disetujui.
- [ ] Target restore adalah project/database terpisah/disposable, bukan production.
- [ ] Output dump disimpan di direktori terenkripsi/terbatas di luar repository.
- [ ] Artefak data dan role tidak dimasukkan ke Git, issue tracker, chat, atau log CI.
- [ ] Target off-site terpisah dari akun/host Supabase utama.
- [ ] Rencana menghapus artefak plaintext setelah enkripsi dan verifikasi tersedia.

## 4) Ekspor dan pengukuran

Untuk Supabase CLI v4 gunakan --remote=<project-ref>. Flag --linked masih digunakan pada CLI v3 tetapi telah deprecated dan dihapus pada v4.

Contoh manual (PowerShell; set project ref saja, jangan tulis password di command/script):

    $ProjectRef = "<SUPABASE_PROJECT_REF>"
    $BackupDir = "D:\simpeg-backups\$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null

    supabase db dump "--remote=$ProjectRef" --file (Join-Path $BackupDir "schema.sql")
    if ($LASTEXITCODE -ne 0) { throw "Schema dump gagal" }

    supabase db dump "--remote=$ProjectRef" --data-only --file (Join-Path $BackupDir "data.sql")
    if ($LASTEXITCODE -ne 0) { throw "Data dump gagal" }

    supabase db dump "--remote=$ProjectRef" --role-only --file (Join-Path $BackupDir "roles.sql")
    if ($LASTEXITCODE -ne 0) { throw "Role dump gagal" }

    Get-ChildItem $BackupDir -File | Get-FileHash -Algorithm SHA256

Helper repo yang menjalankan ekspor tersebut:

    .\scripts\supabase-free-backup.ps1 -ProjectRef "<SUPABASE_PROJECT_REF>" -OutputRoot "D:\simpeg-backups"

Script hanya menghasilkan SQL dan manifest checksum. Script tidak mengenkripsi, mengunggah off-site, mengatur jadwal, atau menghapus artefak lama. Jalankan melalui Task Scheduler/runner yang dikelola, dengan akun yang sama yang memiliki credential Supabase CLI; enkripsi dan unggah file ke media off-site harus menjadi langkah tersendiri.

Catat waktu dan durasi setiap langkah:

| Langkah | Mulai | Selesai | Durasi | Catatan |
|---|---|---|---|---|
| (A) Schema dump | | | | |
| (B) Data dump | | | | |
| (C) Role dump | | | | |
| (D) Encrypt + upload off-site | | | | |
| (E) Verifikasi checksum/artefak | | | | |

**Timestamp artefak yang dipulihkan:** ____________________. Bandingkan dengan waktu insiden hipotetis untuk menaksir RPO. Jangan menyimpulkan RPO 24 jam hanya dari durasi dump yang singkat.

## 5) Restore ke target disposable (wajib)

Supabase CLI menghasilkan SQL plain text. Pada target PostgreSQL disposable, restore menggunakan psql dengan ON_ERROR_STOP; jangan menggunakan pg_restore untuk file plain SQL.

    $env:TARGET_DATABASE_URL = "<TARGET_DISPOSABLE_DATABASE_URL>"

    # Periksa SQL hasil ekspor terlebih dahulu. Jika memakai custom roles,
    # tinjau dan terapkan roles.sql sebelum objek yang bergantung pada role.
    psql "$env:TARGET_DATABASE_URL" -v ON_ERROR_STOP=1 -f ".\roles.sql"
    psql "$env:TARGET_DATABASE_URL" -v ON_ERROR_STOP=1 -f ".\schema.sql"
    psql "$env:TARGET_DATABASE_URL" -v ON_ERROR_STOP=1 -f ".\data.sql"

Jika target adalah project Supabase disposable, tinjau default privileges sebelum menjalankan SQL. Supabase memperingatkan bahwa tabel pada target dapat mewarisi seluruh default privileges target; cabut privilege default yang tidak sesuai sebelum restore schema. Pastikan role platform yang diperlukan sudah ada dan jangan menyalin kredensial production ke target uji.

Jika sumber cadangan diambil dengan format custom pg_dump -Fc (bukan Supabase CLI SQL files), gunakan pg_restore sesuai format tersebut. Catat metode dan versinya.

## 6) Verifikasi pascapulih

- [ ] Restore schema/data berhasil tanpa error fatal; semua warning dicatat dan ditinjau.
- [ ] Schema yang memang diekspor sesuai sumber; schema managed Supabase/extension yang tidak masuk dump dinyatakan sebagai batas cakupan.
- [ ] Goose status/versi sesuai aplikasi.
- [ ] Jumlah tabel, indeks, constraint, RLS, policy, owner, grants, dan default privileges ditinjau.
- [ ] Custom roles yang dibutuhkan tersedia (jika digunakan).
- [ ] Sampel read-only query dan integritas referensial lolos.
- [ ] Backend pada lingkungan uji memakai database hasil restore dan GET /ready menghasilkan 200.
- [ ] Login, RBAC, pembacaan data, dan alur kritis diuji pada lingkungan uji; jangan expose data pegawai ke publik.
- [ ] Objek dokumen Cloudinary diverifikasi melalui prosedur asset terpisah.
- [ ] Artefak plaintext dan disposable project/database dihapus setelah persetujuan.

**Hitungan baris kunci (isi sumber dan hasil restore):**

| Tabel/objek | Sumber | Hasil restore | Cocok? |
|---|---:|---:|---|
| user | | | |
| session | | | |
| account | | | |
| auditLog | | | |
| gaji | | | |
| (domain kritis lain) | | | |

## 7) Hasil & evaluasi

| Metrik | Target awal | Hasil terukur | Status |
|---|---:|---:|---|
| RPO | 24 jam | | ☐ Memenuhi ☐ Tidak ☐ Terbatas tier |
| RTO hingga API siap | 4 jam | | ☐ Memenuhi ☐ Tidak |
| Kelengkapan ekspor schema/data/roles | Semua artefak diperlukan | | ☐ OK ☐ Temuan |
| Keutuhan data & RLS/grants | Sesuai sumber | | ☐ OK ☐ Temuan |
| Enkripsi & off-site | Wajib | | ☐ OK ☐ Belum |
| Jadwal & alert backup | Wajib | | ☐ OK ☐ Belum |

**Temuan & tindak lanjut:**
-
-

**Kesimpulan:** ☐ Lulus lingkup uji ini ☐ Lulus dengan catatan ☐ Gagal/perlu uji ulang

**Penanggung jawab & tanggal tanda tangan:** ______________________________

---

_Lampirkan manifest/checksum dan log waktu tanpa password, connection string, API key, PII, atau isi data. Setelah selesai, ringkas hasil pada docs/RUNBOOK.md/docs/DECISIONS.md. Drill staging/disposable tidak boleh ditulis sebagai bukti restore produksi._
