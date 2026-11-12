# PRD: SIMPEG

- Status: Draft baseline; product decisions below are not yet confirmed.
- Last updated: 2026-11-11
- Product: Sistem Informasi Kepegawaian (SIMPEG)
- Log keputusan ringkas beserta pemilik, tanggal, dan status: `docs/DECISIONS.md`.

## 1. Ringkasan

SIMPEG direncanakan sebagai aplikasi untuk mengelola data pegawai dan proses kepegawaian dalam satu sumber informasi yang dapat dicari, diperbarui, dan diaudit oleh pengguna yang berwenang. Backend dan frontend dibangun sebagai aplikasi terpisah dan berkomunikasi melalui API. Skema PostgreSQL aplikasi dibentuk oleh migrasi Goose di `backend/migrations/`; backend mengakses database dengan model GORM di `backend/internal/models`.

Migrasi yang tersedia memuat domain lebih luas dari MVP awal, termasuk presensi, cuti, penggajian, mutasi, dokumen, kinerja, dan autentikasi. Keberadaan tabel tidak dengan sendirinya menetapkan prioritas rilis atau aturan bisnis final. Kebijakan organisasi, yurisdiksi, dan urutan peluncuran perlu dikonfirmasi.

## 2. Masalah dan tujuan

### Masalah yang ingin diselesaikan

- Data pegawai dan organisasi berpotensi tersebar di berkas atau sistem yang berbeda.
- Petugas membutuhkan cara konsisten untuk mencari dan memperbarui data yang menjadi tanggung jawabnya.
- Perubahan data penting perlu dapat ditelusuri.

Pernyataan masalah ini adalah hipotesis produk dan perlu divalidasi dengan calon pengguna.

### Tujuan MVP

- Menyediakan profil pegawai terpusat dengan identitas pegawai yang unik.
- Mengelola struktur organisasi dan penempatan pegawai.
- Membatasi tindakan berdasarkan peran dan kewenangan data.
- Menyediakan pencarian, filter, dan ringkasan data pegawai yang bermanfaat untuk operasional.
- Menyimpan jejak perubahan untuk data kepegawaian yang sensitif.

### Belum menjadi komitmen MVP

- Schema sudah memodelkan penggajian, pajak, tunjangan, presensi, cuti, lembur, penilaian kinerja, tugas belajar, dan sejumlah proses kepegawaian lain. Modul-modul ini belum otomatis menjadi bagian rilis MVP; prioritas dan aturan bisnisnya perlu dikonfirmasi.
- Rekrutmen dan integrasi eksternal belum ditetapkan dalam schema maupun ruang lingkup MVP.

Ruang lingkup ini dapat diperluas setelah pemilik produk menyetujui prioritas dan aturan bisnisnya.

## 3. Pengguna dan peran (usulan)

- Admin sistem: mengelola akses dan konfigurasi dasar.
- Operator kepegawaian: mengelola data pegawai dan organisasi sesuai kewenangan.
- Atasan/pejabat: melihat data dan ringkasan pegawai dalam cakupan organisasinya.
- Pegawai: melihat profilnya sendiri; kemampuan mengubah data pribadi belum diputuskan.

Nama peran, batas organisasi, dan izin final harus disesuaikan dengan model organisasi yang sebenarnya.

## 4. Ruang lingkup fungsional MVP (usulan)

### FR-01: Akses pengguna

- Usulan login awal adalah Google OAuth tanpa login kata sandi, sesuai konfigurasi yang sudah disiapkan. Pemilik produk harus mengonfirmasi apakah ini kebijakan final.
- Hanya akun yang diizinkan organisasi yang dapat memperoleh akses. Metode allowlist, domain email yang diperbolehkan, undangan, dan provisioning masih perlu diputuskan.
- Sesi harus dikelola di sisi server dengan cookie `HttpOnly`, `Secure` di production, dan perlindungan CSRF/origin yang aktif.
- Pengguna dapat keluar; sesi dapat dicabut saat akun dinonaktifkan atau akses dicabut.
- Tabel `User`, `Session`, `Account`, `Verification`, dan `TwoFactor` sudah ada di schema acuan. Keberadaan tabel tersebut belum berarti alur autentikasi sudah diimplementasikan.
- Resolver sesi API membedakan sesi tidak valid (`401`) dari kegagalan resolver/database (`503`) dan tidak mengirim detail dependency ke klien.
- Fondasi API saat ini menerima `Authorization: Bearer <session-token>` pada `GET /api/me`, memeriksa token pada tabel `session`, dan menolak sesi kedaluwarsa. Ini kontrak sementara untuk API, bukan keputusan provider login; penerbitan token, OAuth callback, logout, dan pencabutan sesi belum dibuat.
- Middleware otorisasi backend menolak secara default dan hanya menerima grant permission eksplisit berdasarkan role server-side. Grant role bawaan telah dikonfirmasi (lihat §11 matriks peran dan cakupan yang disetujui); role atau izin yang tidak tercantum tetap ditolak.

### FR-02: Data pegawai

- Operator berwenang dapat membuat, melihat, memperbarui, dan menonaktifkan catatan pegawai.
- Setiap pegawai memiliki nomor/identitas internal yang unik.
- Profil menampilkan data yang disetujui untuk kebutuhan SIMPEG; daftar field wajib masih perlu disepakati.
- Sistem mencegah duplikasi identitas internal dan memberi pesan validasi yang dapat ditindaklanjuti.

### FR-03: Organisasi dan jabatan

- Pengguna berwenang dapat melihat struktur organisasi.
- Data unit organisasi dan jabatan dapat dikelola oleh peran yang ditetapkan.
- Penempatan pegawai mengacu pada unit dan jabatan yang valid.
- Riwayat perubahan unit/jabatan perlu dipertahankan jika kebijakan mengharuskan pelacakan historis; aturan efektif tanggal masih terbuka.

### FR-04: Pencarian dan daftar

- Pengguna dapat mencari pegawai berdasarkan field yang disetujui.
- Daftar pegawai mendukung filter dan pagination; pengurutan hanya pada kolom yang diizinkan.
- Hasil yang ditampilkan mengikuti kewenangan akses pengguna.

### FR-05: Ringkasan

Dashboard awal dapat menampilkan jumlah pegawai menurut status dan unit organisasi, sepanjang definisi status serta izin akses telah disepakati. Metrik lain tidak termasuk tanpa keputusan produk.

### FR-06: Audit perubahan

Perubahan pada data penting mencatat pelaku, waktu, jenis perubahan, dan referensi catatan yang diubah. Nilai sensitif tidak boleh disalin ke log secara berlebihan. Retensi dan siapa yang dapat membaca audit log perlu diputuskan.

### FR-07: Email pemberitahuan gaji dibayarkan

- Setelah backend menerima dan menyimpan konfirmasi bahwa pembayaran gaji pegawai berhasil, sistem mengantrekan email pemberitahuan kepada pegawai tersebut.
- Penerbitan atau pembuatan slip gaji berstatus `TERBIT` saja tidak boleh memicu email “gaji sudah masuk”.
- Pemrosesan antrean harus idempoten dan dapat dicoba ulang tanpa mengirim duplikat untuk peristiwa pembayaran yang sama.
- Email tidak menyertakan nominal atau data sensitif secara default; email dapat memuat tautan ke halaman slip yang memerlukan login.
- Status pembayaran, sumber konfirmasi, provider email, dan kebijakan kegagalan/pengiriman ulang harus diputuskan sebelum implementasi.

### FR-08: Otorisasi berbasis peran dan cakupan data

- Backend memakai prinsip deny-by-default: setiap endpoint dan aksi yang dilindungi harus memeriksa identitas, izin, dan cakupan data.
- Usulan peran awal: admin sistem, operator kepegawaian, atasan/approver, dan pegawai. Matriks izin per aksi dan cakupan organisasi harus disetujui sebelum implementasi.
- Pegawai hanya boleh mengakses profil dan data miliknya kecuali diberi izin lebih luas. Atasan dibatasi pada cakupan organisasi yang ditetapkan. Operator dan admin hanya mendapat akses sesuai tugasnya.
- Role dan cakupan akses adalah data server-side. Pengguna tidak dapat menaikkan hak melalui profil, request body, metadata yang dapat diedit sendiri, atau manipulasi UI.
- Perubahan role, status akun, dan cakupan akses harus dibatasi untuk admin berwenang dan dicatat dalam audit log.
- Frontend boleh menyembunyikan navigasi yang tidak tersedia, tetapi keputusan akses tetap dilakukan backend.

## 5. Alur utama

1. Operator masuk dan hanya melihat menu yang sesuai dengan aksesnya.
2. Operator mencari pegawai atau membuka daftar pegawai dalam cakupannya.
3. Operator membuat atau memperbarui profil; sistem memvalidasi data dan kewenangan di backend.
4. Setelah perubahan berhasil, frontend menampilkan hasil terbaru dan perubahan tercatat untuk audit.
5. Atasan melihat daftar atau ringkasan sesuai cakupan organisasinya.

## 6. Kebutuhan nonfungsional

- Keamanan: autentikasi dan otorisasi wajib ditegakkan di backend; akses menggunakan prinsip least privilege.
- Privasi: kumpulkan data minimum yang diperlukan, batasi eksposur pada API/UI, dan gunakan data sintetis untuk pengembangan serta pengujian.
- Integritas: identitas pegawai unik; operasi multi-langkah yang harus konsisten dijalankan secara transaksional.
- Keandalan: tampilkan keadaan loading, kosong, berhasil, dan gagal dengan jelas; kegagalan API tidak boleh tampak sebagai keberhasilan.
- Aksesibilitas: alur utama dapat digunakan dengan keyboard dan kontrol memiliki label yang bermakna.
- Keterpeliharaan: backend dan frontend dapat dibangun dan diuji secara terpisah; kontrak API terdokumentasi.
- Kinerja dan ketersediaan: target numerik belum ditetapkan. Tetapkan setelah volume pegawai, pola akses, dan lingkungan deployment diketahui.
- Kepatuhan dan retensi: standar yang berlaku belum diketahui dan tidak boleh diklaim sebelum dikonfirmasi.
- Integrasi eksternal: rahasia hanya berada di backend; pekerjaan asinkron harus aman terhadap retry dan kegagalan sementara.

## 7. Kriteria penerimaan MVP

- Pengguna tanpa autentikasi tidak dapat membaca atau mengubah endpoint yang dilindungi.
- Pengguna tanpa izin tidak dapat membaca atau mengubah data di luar cakupannya, termasuk melalui pemanggilan API langsung.
- Akun yang tidak memenuhi kebijakan allowlist/provisioning tidak dapat membuat sesi SIMPEG.
- Tes otorisasi mencakup akses sendiri, akses lintas pegawai, lintas unit organisasi, perubahan role, dan endpoint administratif.
- Menonaktifkan akun atau mencabut akses membatalkan sesi aktif sesuai kebijakan sesi yang dipilih.
- Operator dapat membuat, mencari, melihat, memperbarui, dan menonaktifkan pegawai sesuai aturan yang dikonfirmasi.
- Nomor pegawai duplikat ditolak dengan respons validasi yang konsisten.
- Penempatan pegawai hanya dapat menggunakan unit dan jabatan yang valid.
- Perubahan data penting dapat ditelusuri tanpa menyimpan rahasia atau menyalin data sensitif secara tidak perlu ke log.
- Frontend menampilkan status loading, kosong, gagal, dan berhasil pada alur data utama.
- Backend dan frontend memiliki pemeriksaan build/typecheck dan pengujian untuk perilaku yang diubah.
- Pemberitahuan gaji hanya dibuat setelah konfirmasi pembayaran tersimpan; pembuatan atau penerbitan slip tidak memicu pemberitahuan.
- Retry notifikasi untuk peristiwa pembayaran yang sama tidak menghasilkan email duplikat, dan kegagalan pengiriman dapat ditelusuri.

## 8. Batas arsitektur dan stack

### Backend

- Bahasa: Go.
- ORM: GORM.
- Database relasional: Supabase PostgreSQL, dengan schema yang dibentuk oleh migrasi Goose di `backend/migrations/`.
- Model Go/GORM di `backend/internal/models` memetakan tabel dan kolom PostgreSQL yang didefinisikan oleh migrasi.
- Naming strategy GORM mempertahankan nama tabel dan kolom yang digunakan migrasi.
- Backend memiliki API HTTP/JSON dan bertanggung jawab atas aturan bisnis, validasi, otorisasi, serta akses data.
- Migrasi produksi harus eksplisit, berversi, dan menjadi sumber DDL; jangan mengandalkan GORM AutoMigrate. Status migrasi aktual harus diverifikasi pada database target.

### Layanan pendukung

- Supabase menyediakan PostgreSQL terkelola. Backend GORM memakai koneksi server-side dengan TLS dan role database berhak minimum. Jika Data API Supabase mengekspos tabel aplikasi, RLS dan kebijakan per pengguna/unit wajib diterapkan.
- Upstash Redis tersedia untuk cache atau state sementara yang use case, TTL, dan fallback-nya disepakati. PostgreSQL tetap sumber kebenaran untuk pegawai, payroll, dan status pembayaran.
- QStash menangani pengiriman pekerjaan asinkron/terjadwal dari backend, termasuk antrean email. Callback harus memverifikasi signature mentah QStash dan idempoten terhadap retry.
- Email dikirim backend setelah status pembayaran gaji terkonfirmasi. Provider dan identitas pengirim belum dipilih. Pesan minimal dan tidak memuat nominal gaji tanpa persetujuan kebijakan.

### Frontend

- TanStack Start untuk aplikasi web.
- shadcn/ui untuk komponen UI.
- TanStack Query untuk server state dan TanStack Store untuk client state bersama.
- TanStack Table untuk tabel data.
- Frontend mengakses backend melalui kontrak API, bukan database.

### Domain yang sudah dimodelkan

- Data pegawai, kategori, unit/homebase, jabatan, pendidikan, dan riwayat gaji.
- Autentikasi, sesi, akun, role, 2FA, delegasi, riwayat login, dan audit log.
- Presensi, mesin dan template sidik jari, cuti, saldo cuti, lembur, dan tugas belajar.
- Penggajian, komponen gaji, slip, pesangon, mutasi, disiplin, pemberhentian, dan dokumen.
- Sasaran kinerja, pengumuman, survei, notifikasi, serta antrean pesan keluar.

Daftar ini merangkum model yang tersedia; detail dan aturan bisnis wajib dibaca dari schema serta dikonfirmasi dengan pemilik produk sebelum dipakai sebagai persyaratan perilaku.

### Keputusan integrasi yang masih terbuka

- Format kontrak dipilih OpenAPI 3.1; draft MVP ada di `docs/openapi.yaml`. URL versioning, lokasi/provider runtime autentikasi, kontrak sesi frontend-backend, deployment, nama modul/repo, tool migrasi, dan mekanisme generate tipe masih terbuka.
- Konfirmasi login Google-only, domain/allowlist akun, provisioning/undangan, masa sesi, kebijakan pencabutan sesi, serta kewajiban MFA untuk peran istimewa.
- Matriks izin per peran dan model cakupan data organisasi.
- Use case, TTL, dan fallback Upstash Redis; konfigurasi koneksi/pool Supabase untuk lingkungan deployment.
- Sumber event/status pembayaran gaji yang dianggap terkonfirmasi. Schema saat ini hanya memiliki status slip `DRAF` dan `TERBIT`, sehingga perlu perubahan schema sebelum FR-07 dapat diimplementasikan.
- Provider email, alamat pengirim/domain, kebijakan retry/retensi, dan apakah pegawai dapat memilih keluar dari email non-wajib.
- Versi dan package manager belum dipilih karena repo belum memiliki manifest.

## 9. Ukuran keberhasilan (perlu baseline)

- Waktu yang dibutuhkan operator untuk menemukan dan memperbarui profil pegawai.
- Persentase profil yang memenuhi kelengkapan field wajib setelah field disepakati.
- Jumlah duplikasi atau koreksi data yang dilaporkan.
- Keberhasilan pengguna menyelesaikan alur kerja utama tanpa bantuan.

Tetapkan definisi, baseline, target, serta cara pengukuran sebelum menggunakan metrik ini sebagai komitmen.

## 10. Asumsi dan pertanyaan terbuka

- Siapa pemilik produk dan siapa pengguna pertama yang akan memvalidasi alur?
- Target institusi dikonfirmasi sebagai Universitas Subang; kebijakan kampus yang berlaku tetap harus diidentifikasi dan disetujui sebelum aturan bisnis diterapkan.
- Data pegawai apa yang wajib, opsional, sensitif, atau dilarang disimpan?
- Apakah pegawai boleh memperbarui profil sendiri? Apakah perubahan perlu persetujuan?
- Bagaimana struktur peran, unit organisasi, dan cakupan akses dibentuk?
- Apakah MVP memerlukan impor/ekspor data, dan format serta proses persetujuannya seperti apa?
- Dari domain yang sudah dimodelkan oleh migrasi dan backend, modul mana yang masuk MVP dan mana yang dirilis pada fase berikutnya?
- Apa sumber identitas/login, kebijakan MFA, dan proses keluar-masuk pengguna?
- Tool migrasi, deployment, backup/restore, retensi audit, target kinerja, dan target ketersediaan apa yang diwajibkan?
- Sistem eksternal apa yang perlu diintegrasikan dan siapa pemilik datanya?

## 11. Draf rekomendasi tahap 0 (belum disetujui)

Status: draf untuk ditinjau pemilik produk per 10 November 2026. Draf ini bukan keputusan atau requirement final kecuali yang dinyatakan sebagai keputusan terkonfirmasi di bawah. Target yang dikonfirmasi adalah Universitas Subang dan MVP awal dibatasi pada tahap 1. Rektor dan superadmin disebut sebelumnya, tetapi peran mereka sebagai pemilik/pilot/approver, kelompok pilot, dan lingkungan deployment belum dikonfirmasi.

**Keputusan terkonfirmasi:** target institusi adalah Universitas Subang; tahap 1 di bawah menjadi MVP awal. Tahapan 2-5 tetap usulan fase lanjutan, bukan komitmen rilis.

### Keputusan fondasi presensi mesin (10 November 2026)

- Pekerjaan fondasi presensi dimulai sekarang atas keputusan pengguna; ini tidak menetapkan aturan bisnis status hadir/pulang atau memindahkan seluruh modul presensi ke MVP awal tahap 1.
- Target perangkat adalah Fingerspot Revo W-230N. Pengguna lebih memilih integrasi API langsung, tetapi dokumentasi API/SDK vendor belum tersedia. Jangan menebak protokol, kredensial, autentikasi perangkat, atau alur sinkronisasi; adapter vendor belum dibuat.
- Simpan fakta tap ternormalisasi yang minimum: ID mesin terdaftar, PIN mesin sebagai kunci pemetaan pengguna eksternal, waktu kejadian dengan offset eksplisit (disimpan UTC dengan presisi milidetik), dan idempotency key dari adapter. PIN belum dipetakan ke `User`/NIP. Event tidak mengubah `Presensi` yang tetap satu status per orang/hari.
- Retry dengan mesin dan idempotency key sama serta PIN/waktu sama mengembalikan event yang sama; key yang sama dengan fakta tap berbeda ditolak sebagai konflik. Event tidak menentukan status harian.
- Rekomendasi non-final setelah mempertimbangkan fingerprint: bila kebutuhan operasional dan kajian privasi menyetujuinya, pencocokan dapat dilakukan pada perangkat saja; template biometrik tidak dikirim atau disimpan di SIMPEG. Ini tetap merupakan pemrosesan biometrik pada perangkat, bukan berarti data biometrik tidak diproses.
- Pemakaian fingerprint belum disetujui untuk pilot/produksi. Sebelum mengaktifkan atau mendaftarkan sidik jari, konfirmasi aliran data perangkat dan Fingerspot.io (termasuk kemungkinan sinkronisasi cloud), kebutuhan dan dasar pemrosesan, pemberitahuan/otorisasi, alternatif presensi non-biometrik, prosedur koreksi saat gagal cocok, serta retensi dan penghapusan template pada mesin/vendor. Jangan aktifkan sinkronisasi template atau simpan template/payload biometrik di SIMPEG tanpa keputusan dan desain keamanan terpisah.
- Tetap dikecualikan dari API event SIMPEG: template sidik jari atau payload biometrik, GPS, foto, payload mentah/vendor-spesifik, dan derivasi status harian.
- Fondasi API menggunakan pola sesi sementara dan permission backend yang sudah ada; permission ingestion baru tidak diberi grant. Autentikasi perangkat/kredensial produksi, grant role, sumber idempotency key yang dijamin vendor, interpretasi zona waktu mesin, migration runner, penerapan migrasi, serta aturan retensi masih belum diputuskan.

### Usulan tahapan ruang lingkup

Pengguna semula memilih seluruh kelompok domain sebagai kandidat. Pada 10 November 2026, pengguna menyetujui tahap 1 saja sebagai MVP awal; tahapan 2-5 tetap usulan fase lanjutan dan belum disetujui.

1. **MVP awal — fondasi dan data inti:** autentikasi/akun, role dan cakupan akses, profil pegawai, kategori, unit, jabatan, pencarian/daftar, ringkasan dasar, serta audit perubahan. Batas tahap ini telah disetujui; rincian kebijakan dan implementasi tetap memerlukan keputusan yang dicatat pada bagian ini.
2. **Operasional kepegawaian:** presensi dasar dan cuti/persetujuan. Perangkat biometrik, geofencing, dan aturan cuti khusus menunggu kebijakan, kebutuhan pilot, serta tinjauan privasi.
3. **Penggajian:** komponen gaji dan slip hanya setelah kebijakan payroll, akses khusus, serta proses verifikasi dan persetujuan disetujui. Schema saat ini hanya memodelkan status slip `DRAF`/`TERBIT`, bukan bukti dana dibayarkan.
4. **Integrasi pemberitahuan:** email otomatis “gaji sudah masuk” hanya setelah sumber konfirmasi pembayaran dan alur outbox/retry/idempotensi disetujui. Pembuatan atau penerbitan slip bukan bukti pembayaran.
5. **Tahap lanjutan:** kinerja, lembur/tugas belajar, mutasi/disiplin/pemberhentian/dokumen, pengumuman, dan survei; prioritaskan setelah kebutuhan pilot tervalidasi.

Batas MVP awal tahap 1 telah disetujui. Dependensi antartahap dan kebijakan institusi tetap harus ditinjau sebelum fitur pada tahap lanjutan menjadi komitmen.

### Matriks peran dan cakupan (disetujui 2026-11-10)

`Role` di schema hanya menyimpan nama/deskripsi role dan relasi ke pengguna; schema tidak memuat grant aksi, hierarki unit, atau cakupan akses. Grant berikut diterapkan di `backend/cmd/server/main.go` sebagai `auth.StaticPolicy` dan ditranslasikan ke permission server-side pada setiap request. Role atau izin yang tidak tercantum dalam matriks ini ditolak (deny-by-default).

| Peran | Permission yang diizinkan | Cakupan dan batas |
|---|---|---|
| admin | `employee:read:any`, `employee:create`, `employee:update:any`, `employee:archive`, `employee:role:manage`, `employee:import`, `organization:read`, `organization:manage`, `attendance:event:ingest`, `attendance:read:any`, `audit:read`, `category:read`, `payroll:read:any`, `payroll:manage`, `payroll:schedule`, `holiday:read`, `holiday:manage` | Seluruh organisasi. Akses ke data payroll atau sensitif tambahan masih memerlukan grant eksplisit dan diaudit. |
| operator | `employee:read:homebase`, `employee:create:homebase`, `employee:update:homebase`, `employee:import:homebase`, `organization:read:homebase`, `organization:manage:homebase`, `audit:read:homebase`, `category:read`, `attendance:read:homebase`, `payroll:read:homebase`, `payroll:schedule:homebase`, `holiday:read` | Unit yang ditugaskan beserta seluruh turunannya pada pohon `Homebase`. Tidak mengelola role; pencatatan konfirmasi pembayaran memerlukan grant terpisah. |
| atasan | `employee:read:homebase`, `organization:read:homebase`, `category:read`, `attendance:read:homebase`, `holiday:read` | Unit tempat ditugasi beserta anak-anaknya. Tidak dapat mengakses unit lain atau mengubah data pegawai. |
| pegawai | `employee:read:self`, `attendance:read:self`, `payroll:read:self` | Hanya profil, riwayat presensi, dan slip gaji miliknya sendiri. |

Permission berakhiran `:any` berarti akses lintas seluruh organisasi dan hanya diberikan kepada `admin`. Permission berakhiran `:homebase` berarti akses dibatasi pada unit penugasan principal beserta turunannya, dan diberikan kepada `operator` serta `atasan`. Permission `audit:read:homebase` memberi operator akses ke jejak perubahan pada unitnya beserta turunannya — bukan jejak global. Permission `category:read` memberi akses baca ke daftar kategori pegawai; hanya kategori aktif yang dikembalikan.

Pembacaan presensi memakai `attendance:read:self` untuk pegawai, `attendance:read:homebase` untuk operator/atasan, dan `attendance:read:any` untuk admin. `GET /api/v1/attendance/my-records` membaca record yang sudah ada pada tabel `Presensi`; `GET /api/v1/attendance/unit-summary` menghitung jumlah record per tanggal/status sesuai cakupan. Keduanya tidak menurunkan status dari `AttendanceEvent`; PIN mesin belum dipetakan ke pegawai dan aturan status harian belum disetujui.

**Cakupan operator ditetapkan 11 November 2026:** cakupan operator adalah unit penugasannya **beserta seluruh turunan** pada pohon `Homebase` (sama dengan atasan). Operator yang ditempatkan di FAKULTAS karena itu dapat mengelola pegawai JURUSAN di bawahnya. Operator membuat unit baru hanya di bawah induk yang sudah berada dalam cakupan; unit akar berada di luar setiap cakupan unit dan hanya dapat dibuat oleh `admin`. Template impor pegawai tidak memuat kolom unit, sehingga pegawai hasil impor oleh operator ditempatkan pada unit penugasan operator.

Backend memeriksa permission pada setiap request. Cakupan organisasi ditegakkan di lapisan aplikasi: `operator` dan `atasan` memakai permission `:homebase` yang dibatasi pada unit penugasan beserta turunannya (recursive CTE pada pohon `Homebase`), `admin` memakai permission `:any` untuk seluruh organisasi, dan `pegawai` memakai `:self`. Principal tanpa penugasan unit tidak melihat data pegawai apa pun. Akses ke gaji dan konfirmasi pembayaran memakai permission payroll khusus: `payroll:read:any` (admin), `payroll:read:homebase` (operator, slip pegawai pada unitnya beserta turunannya), `payroll:read:self` (pegawai, slipnya sendiri), dan `payroll:manage` (admin, mencatat konfirmasi pembayaran). Jadwal tanggal gajian per unit memakai `payroll:schedule` (admin, seluruh unit) dan `payroll:schedule:homebase` (operator, unitnya beserta turunannya). Kalender hari libur nasional memakai `holiday:read` (admin, operator, atasan) dan `holiday:manage` (admin; memicu penyegaran dari penyedia).

**Aturan eviden pengajuan cuti (disetujui 12 November 2026):** cuti melahirkan (`BERSALIN`), cuti sakit (`SAKIT` dan `SAKIT_BERKEPANJANGAN`), dan cuti harian (`IZIN_PENTING`) wajib menyertakan eviden saat pengajuan dibuat; pengajuan tanpa eviden ditolak dengan `422`, dan eviden dicek ulang saat persetujuan. Bentuk eviden dibedakan: `SURAT_DOKTER` (berkas, nomor, dan tanggal surat dokter wajib) hanya boleh dipakai cuti sakit dan melahirkan; `FOTO` (berkas dan waktu tangkap wajib) atau `LAINNYA` (waktu dan keterangan wajib) untuk cuti harian. Metadata eviden disimpan pada kolom `jenisEviden`, `fileEvidenUrl`, `waktuEviden`, dan `catatanEviden` tabel `Cuti`; kolom lama `fileSuratUrl` diubah nama menjadi `fileEvidenUrl` lewat migrasi `000019_cuti_eviden.sql`.

**Manajemen dokumen pegawai/dosen (disetujui 12 November 2026):** setiap pegawai/dosen memiliki arsip dokumen digital pada tabel `arsipDokumenPegawai` (endpoint `GET/POST/PATCH/DELETE /api/v1/arsip{...}`). Berkas diunggah langsung ke Cloudinary oleh klien frontend; backend hanya menyimpan URL dan metadata berkas (`fileUrl`, `fileSize`, `mimeType`, `cloudinaryPublicId`) tanpa menangani isi berkas. Setiap jenis dokumen memiliki aturan format MIME dan ukuran maksimal:

| Jenis dokumen | Wajib/Opsional | Format yang diizinkan | Ukuran maksimal |
|---|---|---|---|
| Foto (`FOTO`) | Wajib | image/png, image/jpeg, image/webp | 3.1 MB |
| KTP (`KTP`) | Wajib | image/png, image/jpeg, image/webp, application/pdf | 1.1 MB |
| Kartu Keluarga (`KARTU_KELUARGA`) | Wajib | image/png, image/jpeg, image/webp, application/pdf | 1.1 MB |
| Ijazah Terakhir (`IJAHAN_TERAKHIR`) | Wajib | application/pdf | 5.1 MB |
| SK Kepegawaian/Kontrak (`SK_KEPEGAWAIAN_KONTRAK`) | Wajib | application/pdf | 7.1 MB |
| NPWP (`NPWP`) | Opsional | image/png, image/jpeg, image/webp, application/pdf | 1.1 MB |
| BPJS Ketenagakerjaan (`BPJS_KETENAGAKERJAAN`) | Opsional | image/png, image/jpeg, image/webp, application/pdf | 1.1 MB |
| BPJS Kesehatan (`BPJS_KESEHATAN`) | Opsional | image/png, image/jpeg, image/webp, application/pdf | 1.1 MB |
| Sertifikat Serdos (`SERTIFIKAT_SERDOS`) | Opsional | application/pdf | 5.1 MB |

Konfigurasi Cloudinary dibaca dari environment `CLOUDINARY_CLOUD_NAME`, `CLOUDINARY_API_KEY`, dan `CLOUDINARY_API_SECRET` (lihat `.env.example`). Backend menyediakan endpoint `GET /api/v1/cloudinary/upload-signature` (permission `arsip:create`) yang menghasilkan tanda tangan unggah (SHA-1 atas parameter terurut + API secret) beserta `cloudName`, `apiKey`, dan `timestamp` untuk klien. Nomor rekening (`BankRekening`) dan nama rekening (`BankAtasNama`) sudah tersedia sebagai kolom pada tabel `user`. Timestamps `createdAt`/`updatedAt` disimpan UTC di database dan ditampilkan Asia/Jakarta melalui `internal/timeutil`. Migrasi `000020_arsip_dokumen_tambahan.sql` menambah jenis dokumen baru dan kolom metadata berkas.

**Catatan implementasi (11 November 2026):** batas operator "hanya unit yang ditugaskan" kini ditegakkan — sebelumnya `operator` memegang `employee:read:any` dan `employee:update:any` yang bersifat global. Verifikasi terjadi pada route dan query: daftar pegawai difilter dengan `ScopeHomebaseIDs`, dan setiap record diperiksa per-pegawai pada use case baca/ubah/arsip. Cakupan organisasi untuk `organization:read` dan `organization:manage` juga sudah dibatasi per unit.

**Perbedaan kebijakan dan implementasi yang wajib diketahui agen/pengembang:** matriks di atas adalah kebijakan yang disetujui; konfigurasi runtime berada di `backend/cmd/server/main.go`. Role `atasan` (`employee:read:homebase`) dan `pegawai` (`employee:read:self`) memakai endpoint daftar/detail pegawai: route GET menerima ketiga permission tersebut dan use case mempersempit hasil ke unit penugasan beserta turunannya (atasan/operator) atau ke record sendiri (pegawai). Permission `employee:import` diberikan kepada `admin`; `employee:import:homebase` kepada `operator`. Cakupan unit untuk `organization:read` dan `organization:manage` sudah diterapkan. Jangan menyatakan seluruh cakupan sudah ditegakkan atau memperluas grant tanpa memverifikasi route dan query.

### Keputusan autentikasi dan kontrak sesi (disetujui 2026-11-10)

**Provider/runtime:** Better Auth. Migrasi `000001_directory_crud.sql` menyediakan tabel bergaya Better Auth (`session`, `account`, `verification`, `twoFactor`); konfigurasi Google OAuth sudah tersedia di `.env.example`.

**Kontrak:** sesi server dikelola dengan cookie `HttpOnly`, `Secure` di production, `SameSite` yang sesuai, trusted-origin, dan CSRF protection. Go API memverifikasi sesi terhadap sumber server-side yang disepakati, memeriksa akun aktif serta permission/cakupan. Jangan percaya role atau identity header dari browser. Bila frontend dan API memakai host berbeda, topology cookie/proxy harus ditetapkan dan diuji. Jangan mengekspos token sesi ke penyimpanan browser atau menyamakan route guard dengan otorisasi.

Sebelum produksi, pemilik harus mengonfirmasi: Google Workspace institusi sebagai provider login yang diizinkan, metode allowlist/invitasi/provisioning, masa sesi, pencabutan, dan MFA (disarankan wajib bagi admin serta pemegang permission payroll). Sesi, logout, revokasi, akun nonaktif, dan CSRF harus diuji end-to-end.

### Status implementasi tabel autentikasi

Empat tabel pada migrasi awal dipetakan ke model GORM di `backend/internal/models` dan dipakai sebagai berikut. Status di bawah menyatakan kondisi kode saat ini, bukan kelengkapan alur login.

| Tabel acuan | Kolom kunci | Model GORM | Status implementasi |
|---|---|---|---|
| `session` | `id`, `userId`, `expiresAt`, `token` (unik), `ipAddress`, `userAgent` | `models.Session` | **Dipakai.** `GormSessionResolver` mencari token, menolak sesi kedaluwarsa, dan mengambil `userId` untuk lookup pengguna. `GormSessionRevoker` menghapus baris saat logout. |
| `account` | `id`, `userId`, `providerId`, `accountId`, token OAuth, `password` (credential provider) | `models.Account` | **Dipetakan saja, belum dipakai.** Tidak ada alur callback OAuth atau login kata sandi di backend; tabel akan dikelola runtime Better Auth. |
| `verification` | `id`, `identifier`, `value`, `expiresAt` | `models.Verification` | **Dipetakan saja, belum dipakai.** Tidak ada penerbitan token verifikasi, reset kata sandi, atau verifikasi email. |
| `twoFactor` | `id`, `secret`, `backupCodes`, `userId` (unik), `verified`, `failedVerificationCount`, `lockedUntil` | `models.TwoFactor` | **Dipetakan saja, belum dipakai.** Tidak ada pendaftaran TOTP, verifikasi MFA, atau penguncian akun. Kolom `twoFactorEnabled` pada `User` juga belum dipakai backend. |

Catatan penting:

- **Belum ada penerbitan sesi.** Backend hanya memverifikasi sesi yang sudah ada; tidak ada endpoint login, callback OAuth, atau pendaftaran akun. Frontend (TanStack Start) belum dibuat, sehingga runtime Better Auth belum ada.
- **Logout tersedia untuk sesi opaque saja.** `POST /api/auth/logout` menghapus baris `session`. Penrevokasian token JWT milik Better Auth harus dilakukan oleh runtime autentikasi, bukan oleh backend Go.
- **Perubahan role berlaku pada permintaan berikutnya.** Role dibaca dari tabel `Role` pada setiap resolusi sesi, sehingga perubahan role tidak memerlukan penerbitan sesi baru. Akun nonaktif (`aktif = false`) langsung ditolak sebagai sesi tidak valid.
- **Pencabutan sesi sama dengan penghapusan baris `session`.** Sesudahnya token tidak lagi berlaku dan permintaan berikutnya menerima `401`.
- **`account` menyimpan kredensial OAuth dan hash kata sandi.** Jangan pernah menulis nilai `accessToken`, `refreshToken`, `idToken`, atau `password` ke log, respons API, atau dokumentasi.

### Implementasi awal CRUD pegawai dan organisasi

Sebagai keputusan implementasi untuk melanjutkan coding (bukan pengesahan kebijakan organisasi), tipe `UNIT`, `FAKULTAS`, `JURUSAN`, `BIRO`, dan `BAGIAN` dipetakan ke satu pohon `Homebase` dengan parent opsional. Hubungan induk-anak fleksibel, tanpa mengunci aturan hierarki kampus. Profil pegawai dan organisasi memakai arsip/nonaktif, bukan hard delete; akun nonaktif langsung ditolak oleh resolver sesi. CRUD hanya mengubah field profil yang diizinkan dan menulis audit aktor/aksi/sasaran tanpa menyalin isi data pribadi.

Endpoint menggunakan pemeriksaan permission eksplisit dan menolak permission yang tidak diberi grant. Grant role dikonfigurasi di `backend/cmd/server/main.go` (lihat matriks di §11). Cakupan unit ditegakkan di lapisan aplikasi: daftar difilter lewat `listing.Options.ScopeHomebaseIDs`, setiap record diperiksa per-pegawai atau per-unit, pegawai hasil impor operator ditempatkan pada unit penugasan operator, dan operator hanya dapat membuat unit di bawah induk yang sudah berada dalam cakupan. Pembuatan profil tidak membuat sesi, menetapkan role, atau mengirim undangan.

Migration SQL awal tersedia di `backend/migrations/000001_directory_crud.sql`, tetapi belum diterapkan. Migration runner sudah dipilih (Goose, `DEC-007`), sedangkan workflow deployment belum dibuat. Tinjau migration dan selesaikan proses penerapannya sebelum memakai versi skema baru.

### Keputusan integrasi layanan pendukung (11 November 2026)

Mekanisme integrasi diimplementasikan lebih dulu; penggunaan nyata mengikuti fitur yang membutuhkannya.

- **Outbox persisten di PostgreSQL** (`backend/internal/outbox`) adalah fondasi. Transaksi bisnis menulis baris outbox dalam transaksi yang sama sehingga pekerjaan tidak hilang saat queue eksternal gagal. Klaim baris memakai `FOR UPDATE SKIP LOCKED` agar tidak diproses dua worker. Kunci idempotensi `dedupeKey` mencegah satu pekerjaan logis terkirim dua kali. Retry memakai backoff eksponensial yang dibatasi; setelah jumlah percobaan maksimum pesan diletakkan pada status `FAILED` sebagai dead-letter yang dapat ditinjau dan diulang kembali. Meja `outboxMessage` dibuat lewat migrasi `000006_outbox_message.sql`.
- **QStash** (`backend/internal/qstash`) dipakai sebagai penjadwal pekerjaan asinkron, bukan sumber pekerjaan. Callback diverifikasi signature HS256 terhadap **raw request body** sebelum payload dipercaya, dengan dukungan rotasi kunci (kunci saat ini dan berikutnya). Subjek tujuan callback, masa berlaku, dan isi body diperiksa. **Endpoint callback yang terdaftar:** `POST /api/v1/internal/qstash/audit/cleanup` — endpoint ini tidak memakai sesi bearer, pengesahnya signature QStash, dan pekerjaannya (retensi jejak audit) bersifat idempoten sehingga percobaan ulang QStash tidak mengubah hasil. Jadwal harian didaftarkan lewat `internal/jobs.EnsureSchedule` yang idempoten terhadap tujuan yang sama.
- **Email** (`backend/internal/notify`) memakai SMTP sesuai konfigurasi pada `README.md` dan `.env.example`. Provider **disetujui** pada 11 November 2026, dengan alamat pengirim `no-reply@simpeg.unisubang.ac.id` dan origin aplikasi `https://simpeg.unisubang.ac.id`. Notifikasi gaji dibangun oleh `notify.SalaryPaidNotice` yang **tidak menerima parameter nominal** — nominal hanya tersedia di halaman terautentikasi dan dirujuk lewat tautan. `notify.SalaryNoticeSender` menolak mengirim bila konfirmasi pembayaran belum tersimpan.
- **Pengiriman lewat outbox** (`notify.EmailDeliverer`) menjembatani antrean keluar persisten ke SMTP: bisnis menulis baris outbox dalam transaksinya, worker mengambilnya dan mengirim. Kesalahan permanen (payload rusak, jenis tidak dikenal, sumber konfirmasi pembayaran belum dikonfigurasi) langsung menjadi dead-letter tanpa diulang; kegagalan sementara dijadwalkan ulang dengan backoff. **Sumber konfirmasi pembayaran sudah ada:** modul penggajian menyimpan bukti pada tabel `pembayaran` yang menempel pada slip gaji dan menandainya `TERBAYAR`; `notify.EmailDeliverer.Confirmations` memakai `payroll/app.Service.ConfirmedPayment`.
- **Redis** (`backend/internal/cache`) hanya untuk cache data referensi non-sensitif dengan TTL pendek dan satu namespace. Lapisan cache menegakkan aturan secara runtime: kunci yang menyerupai sesi, token, kredensial, penggajian, atau status pembayaran ditolak, dan `ReadThrough` selalu memuat ulang dari sumber utama bila cache miss atau Redis gagal. **Use case yang disetujui: cache daftar kategori pegawai** — data referensi berkardinalitas rendah yang dibaca setiap kali formulir pegawai dibuka. TTL 5 menit, namespace `simpeg`, dan kunci memuat seluruh parameter daftar. **Queue tidak wajib untuk alur kritis** karena durability dijamin outbox di PostgreSQL, bukan oleh Redis.
- **S3-compatible storage** belum masuk scope. Mekanismenya tidak dibuat agar tidak ada kode tanpa pemakaian.

### Keputusan stack frontend (11 November 2026)

- **Package manager dan script runner:** Bun. Lockfile yang dipakai adalah `bun.lock`; `package-lock.json`, `yarn.lock`, dan `pnpm-lock.yaml` tidak boleh masuk repo. Perintah Vite dijalankan lewat `bun --bun run dev|build|preview` agar Bun (bukan Node) yang mengeksekusi CLI.
- **Scaffolding:** `bunx @tanstack/cli@latest create` dengan TanStack Start + React dan default yang direkomendasikan (Tailwind CSS dan alias `@/*` dikonfigurasi oleh scaffolder). Add-on shadcn **tidak** dipilih pada tahap ini.
- **shadcn/ui diinisialisasi sebelum menulis komponen atau halaman** agar `cn()`, variabel CSS tema, alias path, dan dependency dasar sudah tersedia. Perintah: `bunx shadcn@latest init -t start` lalu `bunx shadcn@latest add <component>`. `components.json` memakai `style` `new-york`, `cssVariables` true, dan lima alias `components`/`ui`/`lib`/`hooks`/`utils`. `style`, `baseColor`, dan `cssVariables` tidak dapat diubah setelah init.
- **shadcn/ui di frontend adalah satu-satunya sumber komponen.** Komponen ditambahkan lewat CLI lalu diedit langsung di repo; tidak menambah pustaka komponen kedua atau design system kustom di atasnya.
- **TanStack Query** untuk server state. `QueryClient` dibuat di dalam `getRouter()` (bukan di module scope) karena Start membuat satu router per request SSR; client di module scope berisiko menyajikan data cache milik request lain.
- **TanStack Table** untuk tabel data dengan pagination/sorting/filtering di sisi server (`manualPagination`, `manualSorting`, `manualFiltering`), `rowCount` dari server, dan reset `pageIndex` saat sort/filter berubah.
- **TanStack Store** hanya untuk client state yang benar-benar dipakai lintas area UI. Pembacaan di React memakai `useSelector` (`useStore` sudah deprecated).
- **Rincian operasional** tersedia di `agents/rules/frontend.md`.

### Kebijakan jejak audit (11 November 2026)

- **Aksi yang dicatat:** pembuatan, perubahan, dan pengarsipan profil pegawai; pembuatan, perubahan, dan pengarsipan unit organisasi; impor massal pegawai; ingestion event presensi; pencabutan sesi; dan pembacaan jejak audit itu sendiri. Jejak hanya memuat aktor, aksi, sasaran, keterangan minimal, unit cakupan, alamat IP, dan waktu — isi profil pegawai tidak pernah disalin.
- **Jejak bersifat append-only.** Backend tidak memiliki jalur hapus atau ubah pada tabel `auditLog` dari kode aplikasi (dijamin oleh tes pemindaian source). Penghapusan hanya dilakukan melalui `audit.Cleaner` yang menjalankan `DELETE` pada entri yang lebih lama dari `AUDIT_RETENTION_DAYS` hari. Default retensi adalah 90 hari. Jika `AUDIT_RETENTION_DAYS` diatur ke 0, tidak ada pemotongan otomatis.
- **Akses dibatasi per unit.** Setiap entri menyimpan `unitId` — unit cakupan saat aksi terjadi. `admin` membaca seluruh jejak; `operator` hanya membaca jejak unit penugasannya beserta turunannya. Kolom `unitId` diselaraskan pada model GORM dan migrasi `backend/migrations/000003_audit_log_unit.sql`.
- **Pembacaan jejak diaudit.** Setiap pembacaan yang mengembalikan entri mencatat aksi `audit.accessed`. Pembacaan yang tidak mengembalikan entri tidak dicatat agar tabel tidak membengkak oleh request kosong.
- **Jejak role sudah diterapkan.** `PUT /api/v1/employees/{id}/role` memerlukan `employee:role:manage` (admin), hanya menerima role `admin`, `operator`, `atasan`, atau `pegawai`, serta menulis perubahan `user.roleId` dan aksi `employee.role.updated` dalam satu transaksi.
- **Jejak pembayaran sudah ada.** Pencatatan konfirmasi pembayaran menulis aksi `payroll.payment.confirmed`.
- **Arsip dan ekspor jejak.** Ekspor jejak audit tersedia melalui `GET /api/v1/audit-logs/export` (disetujui 12 November 2026, `DEC-018`): berkas CSV dengan BOM UTF-8, urutan kronologis naik, tanpa batas jumlah baris, filter hanya `aksi` dan `search` (sama seperti endpoint daftar), dan setiap ekspor dicatat sebagai aksi `audit.exported`. `admin` (`audit:read`) mengekspor seluruh jejak; `operator` (`audit:read:homebase`) hanya jejak unit penugasannya beserta turunannya. Kewajiban arsip ke media lain dan prosedur pemangkasan di luar retensi otomatis belum ditetapkan. Retensi otomatis (90 hari) dan pekerjaan terjadwalnya sudah disetujui.

### Keputusan versi API (11 November 2026)

- **Strategi:** versi pada prefix path — seluruh endpoint berada di `/api/v1/...`. Endpoint operasional (`/health`, `/ready`) tidak berversi karena bukan kontrak data.
- **Konsekuensi kompatibilitas:** perubahan additive (field baru, endpoint baru, enum baru yang tidak merusak klien) tetap pada `v1`. Versi naik hanya bila ada perubahan yang merusak kompatibilitas (field dihapus/diubah makna, endpoint dipindah, bentuk error berubah). Versi baru disajikan pada prefix terpisah (`/api/v2/...`) dan versi lama tetap didukung selama masa transisi.
- **Penegakan:** `httpserver.VersionGuard` menolak path `/api/...` tanpa versi dengan `400 unversioned_path` beserta path yang benar. Path dengan versi yang tidak dikenal (mis. `/api/v2/...` saat ini `v1`) menghasilkan `404 not_found`.
- **Observability:** setiap respons memuat header `X-API-Version` berisi versi yang dipakai.
- **Pengecualian yang belum disepakati:** versioning berbasis media type (`Accept: application/vnd.simpeg.v1+json`) dan negosiasi versi melalui header tidak dipakai. Pendekatan ini menambah kompleksitas pada klien tanpa manfaat nyata untuk API internal institusi.

### Keputusan environment deployment (11 November 2026)

- **Environment pertama:** Supabase PostgreSQL production, diputuskan oleh pengguna.
- **Pemilik produk:** Rektor Universitas Subang. **Kelompok pilot:** admin sistem dan operator kepegawaian dari satu fakultas dan satu biro. **Pembagian peran approval:** matriks peran disetujui bersama Rektor dan Kabiro. Rincian lengkap ada di `docs/DECISIONS.md` (`DEC-017`).
- **Prasyarat yang belum terpenuhi sebelum deployment production:** migration `000001_directory_crud.sql` dan `000002_attendance_events.sql` belum pernah diterapkan pada database nyata, migration runner (Goose) belum memiliki workflow deployment, dan backup/restore belum diuji. Deployment ke production tidak boleh dilakukan sebelum hal-hal tersebut selesai dan diverifikasi.
- **Belum ditetapkan:** nama perorangan untuk admin sistem, operator, dan anggota Kabiro. Yang dibutuhkan agar pilot dapat berjalan adalah daftar akun beserta peran (`admin`, `operator`) dan unit penugasannya.

### Draf migrasi, backup, audit, dan ketersediaan

- **Migrasi:** Goose (tool migrasi SQL berversi yang dikelola aplikasi Go), dengan file SQL di `backend/migrations/` sebagai sumber DDL. Jangan gunakan GORM `AutoMigrate` untuk production. Penerapan migrasi dan pemeriksaan kesesuaian terhadap database target tetap wajib diverifikasi.
- **Backup/restore:** gunakan kemampuan backup/PITR Supabase sesuai tier yang dipilih, dokumentasikan siapa yang dapat memulihkan, dan uji restore berkala di lingkungan terpisah. Tier, frekuensi/retensi backup, RPO, dan RTO harus ditentukan berdasarkan dampak layanan dan anggaran. **Target telah ditetapkan:** `RPO_HOURS` (default 24 jam), `RTO_HOURS` (default 4 jam), `SLO_TARGET` (default 99.9%) dikonfigurasikan melalui environment variable dan dibaca oleh `config.Load()`.
- **Audit:** batasi isi log pada aktor, aksi, sasaran, waktu, dan metadata minimum; jangan salin rahasia atau isi data pribadi sensitif. Akses audit dibatasi dan diaudit. Retensi serta kewajiban arsip harus mengikuti kebijakan institusi yang dikonfirmasi, bukan angka asumsi.
- **Ketersediaan:** jangan menjanjikan SLO/angka uptime sebelum deployment, tier layanan, jam operasional, RPO/RTO, dan jalur eskalasi dipilih.

### Draf konfirmasi pembayaran dan integrasi pendukung

- **Sumber konfirmasi pembayaran (disetujui 2026-11-10):** rekonsiliasi/settlement bank atau sistem keuangan resmi yang dapat ditelusuri. Persetujuan pejabat dan penerbitan slip saja tidak cukup. Catat referensi batch/transaksi, waktu, dan pelaku pengesahan tanpa menyimpan kredensial bank. Email "gaji sudah masuk" hanya boleh dikirim setelah backend mencatat konfirmasi pembayaran sebagai terkonfirmasi, bukan saat slip baru diterbitkan. Pada MVP rekaman tersebut dimasukkan manual oleh admin ke tabel `pembayaran` (`DEC-026`); adapter integrasi bank menulis lewat jalur yang sama setelah kontrak vendor tersedia.
- **Redis:** rekomendasi untuk MVP adalah tidak mengandalkan Redis sampai ada use case konkret. Bila disetujui kelak, batasi pada cache data referensi non-sensitif dengan TTL pendek (misalnya lima menit sebagai nilai usulan), strategi invalidasi, dan fallback ke PostgreSQL saat Redis gagal. Jangan cache sumber kebenaran, sesi, payroll, atau status pembayaran.
- **Queue:** bila email otomatis masuk scope, gunakan outbox persisten di PostgreSQL dan queue hanya sebagai mekanisme delivery. Transaksi bisnis tidak boleh hilang saat queue gagal; retry harus idempoten dan status kegagalan dapat ditinjau/redrive. QStash adalah kandidat yang sudah tercatat di arsitektur, tetapi belum dinyatakan wajib.
- **Email:** provider dan domain pengirim belum diputuskan. Schema menyebut Resend serta pengiriman manual/draf admin/no-op bila credential tidak tersedia, sedangkan README mendokumentasikan konfigurasi SMTP dan FR-07 mengusulkan notifikasi setelah pembayaran. Perbedaan ini harus diselesaikan sebelum implementasi. Rekomendasi isi: pemberitahuan generik dengan tautan ke halaman terautentikasi, tanpa nominal/slip sebagai lampiran; batasi retry, catat hasil, dan cegah duplikasi berdasarkan event pembayaran.

### Keputusan mata uang, zona waktu, dan struktur backend (11 November 2026)

- **Mata uang:** sistem memakai satu mata uang, Rupiah Indonesia (IDR). Tidak ada konversi antar-mata uang dan tidak ada nilai tukar yang disimpan. Nominal pada database adalah angka desimal dalam Rupiah penuh; representasi teksnya dibentuk hanya oleh `internal/money` dengan konvensi Indonesia (pemisah ribuan titik, desimal koma, awalan `Rp`), misalnya `Rp 1.500.000`. Simbol atau kode mata uang lain tidak boleh muncul di API maupun dokumen.
- **Zona waktu:** zona waktu operasional adalah **Asia/Jakarta (UTC+7, WIB)**. Penyimpanan timestamp memakai UTC — kolom `timestamp without time zone` yang dibuat migrasi menyimpan jam dinding UTC, dan `database.OpenPostgres` memaksa parameter runtime `timezone=UTC` agar default `now()` dan nilai yang ditulis GORM memakai basis yang sama. Tampilan ke pengguna dan aturan kalender (periode penggajian, tanggal pembayaran, tanggal cuti) memakai Asia/Jakarta lewat `internal/timeutil`. Kolom tanggal (`date`) menyimpan kalender tanpa jam dan ditulis sebagai tengah malam UTC agar tanggal yang dibaca kembali tetap sama setelah dikonversi.
- **Struktur backend:** backend dibagi per area bisnis dengan empat lapis yang dapat dibedakan: `domain` (aturan dan nilai tanpa I/O), `app` (use case yang memeriksa permission dan cakupan unit), `repository` (port persistensi plus adapter GORM), dan `httpserver` (transport, DTO, middleware). Area direktori memakai `internal/directory/{domain,app,repository}`; area penggajian memakai `internal/payroll/{domain,app,repository}`. Kontrak daftar bersama (pagination, pencarian, penyaring cakupan, pengurutan) ada di `internal/listing`. Handler tidak boleh memuat aturan bisnis.

### Keputusan jadwal gajian, pemberitahuan slip H+1, dan kalender hari libur (11 November 2026)

- **Jadwal gajian per unit:** tanggal gajian bersifat tetap per bulan, antara 1 dan 28, diinput per unit organisasi — bukan per periode. Nilai disimpan di tabel `jadwalGaji` (satu baris per unit, `unitId` unik, foreign key ke `Homebase`). `GET /api/v1/payroll/schedules` memakai permission `payroll:schedule` (admin, seluruh unit) atau `payroll:schedule:homebase` (operator, unit dalam cakupan penugasannya beserta seluruh turunannya). `PUT /api/v1/payroll/schedules/{unitId}` menyimpan nilai `tanggalGajian` (1–28); operator hanya boleh mengatur unit dalam cakupannya. Perubahan jadwal ditulis bersama jejak audit `payroll.schedule.updated` dalam satu transaksi. Skema tabel ada pada migrasi `backend/migrations/000009_payroll_schedule.sql` dan dipetakan oleh model GORM.
- **Pemberitahuan slip gaji H+1:** slip berstatus `TERBIT` atau `TERBAYAR` pada unit yang punya jadwal gajian diberi pemberitahuan **satu hari kalender setelah tanggal gajian**. Tanggal pemberitahuan dipakai tetap — tidak digeser karena jatuh pada Sabtu, Minggu, atau hari libur nasional, dan tidak maju bila periode berjalan belum lewat jadwalnya. Pemberitahuan adalah pemberitahuan **ketersediaan slip**, bukan bukti pembayaran: isi pesan hanya nama, periode, dan tautan ke halaman penggajian — tidak pernah memuat nominal gaji dan tidak pernah menyebut gaji sudah dibayar. Jenis pesan `email.slip-tersedia` disimpan di outbox dengan kunci idempotensi `slip-available:{slipId}` sehingga setiap slip hanya diberi satu pemberitahuan meski pekerjaan dijalankan berulang. Pengiriman dijalankan sebagai job QStash `payroll.notify` (`POST /api/v1/internal/qstash/payroll/notify`, cron `5 19 * * *` = 02:05 WIB, 3 percobaan ulang) yang menulis jejak `payroll.notified`. Notifikasi gaji ("gaji sudah masuk") tetap terpisah dan hanya boleh terkirim setelah konfirmasi pembayaran tersimpan.
- **Kalender hari libur nasional:** backend mengintegrasikan penyedia **api.co.id** (endpoint `GET /holidays/indonesia`, autentikasi header `x-api-co-id`) untuk mengisi tabel `HariLibur`. Sinkronisasi hanya berjalan bila `HOLIDAY_API_URL` dan `HOLIDAY_API_KEY` terisi, dipicu admin lewat `POST /api/v1/holidays/{year}/refresh` (permission `holiday:manage`); pembacaan kalender (`GET /api/v1/holidays?year=`, permission `holiday:read`, default tahun berjalan WIB) selalu dari tabel lokal dan tidak pernah bergantung pada penyedia. Entri penyedia disimpan bila `type`-nya `Public Holiday`, `National Holiday`, atau `Joint Holiday`; entri `Observance` dilewati, tanggal di luar tahun yang diminta dan tanggal ganda dibuang, dan entri tanpa nama atau dengan tanggal tidak sah menolak seluruh penyegaran. Penyimpanan tahun berjalan dilakukan dengan mengganti seluruh entri tahun itu dalam satu transaksi bersama jejak audit `holiday.refreshed`. Pembacaan kalender di-cache di Redis (`holiday:{year}`, TTL 24 jam) bila Redis dikonfigurasi. Kalender hari libur **tidak** mengubah tanggal gajian maupun pemberitahuan H+1. Kunci API disimpan di `.env`/`.env.example` (`HOLIDAY_API_KEY`) dan tidak pernah dicatat pada jejak audit maupun log. Tabel `HariLibur` dibawa migrasi `backend/migrations/000010_holiday.sql`.
- **Permission baru:** admin memperoleh `payroll:schedule`, `holiday:read`, `holiday:manage`; operator memperoleh `payroll:schedule:homebase`, `holiday:read`; atasan memperoleh `holiday:read`. Matriks ini sudah diterapkan di `backend/cmd/server/main.go`.

### Keputusan modul penggajian dan konfirmasi pembayaran (11 November 2026)

- **Representasi pembayaran terkonfirmasi:** migrasi `backend/migrations/000008_payment_confirmation.sql` menambahkan status slip `TERBAYAR` dan tabel `pembayaran` yang menyimpan referensi transaksi, waktu konfirmasi, pelaku pengesahan, dan keterangan. Konfirmasi dibaca dari baris `pembayaran` yang menempel pada slip, sehingga bukti pembayaran selalu dapat ditelusuri; pemetaan GORM berada di `backend/internal/models`.
- **Pencatatan pembayaran:** `POST /api/v1/payroll/slips/{payrollId}/payment` (permission `payroll:manage`, hanya `admin`). Slip hanya dapat dibayar satu kali dan hanya dari status `TERBIT`. Pencatatan baris `pembayaran`, perubahan status slip, antrean notifikasi gaji, dan jejak audit `payroll.payment.confirmed` ditulis dalam satu transaksi — sehingga notifikasi gaji tidak pernah ada tanpa bukti pembayaran yang tersimpan.
- **Konfirmasi manual pada MVP, adapter bank disiapkan:** integrasi langsung ke bank sulit dilakukan sekarang, sehingga pencatatan pembayaran pada MVP dilakukan manual oleh pemegang `payroll:manage`: admin mengisi referensi transaksi/rekapan bank, waktu, dan pengesah secara telusuri. Integrasi otomatis ke bank (REST dan websocket) disiapkan sebagai adapter lanjutan yang menulis bukti pembayaran lewat jalur transaksional yang sama (tabel `pembayaran`) dengan idempotensi; implementasinya menunggu kontrak API vendor dan tidak dibuat sebelum benar-benar dipakai.
- **Pembacaan slip gaji:** `GET /api/v1/payroll/slips` memakai permission `payroll:read:any` (admin), `payroll:read:homebase` (operator, slip pegawai pada unitnya beserta turunannya), atau `payroll:read:self` (pegawai, slipnya sendiri). Seluruh nominal dikembalikan sebagai Rupiah yang sudah diformat; periode ditulis sebagai `YYYY-MM` beserta label Bahasa Indonesia.
- **Email gaji kini dapat terkirim** karena sumber konfirmasi pembayaran sudah ada. Notifikasi tetap tidak pernah memuat nominal dan hanya dikirim setelah konfirmasi tersimpan.



1. Tetapkan pemilik produk, peran Rektor/superadmin dalam keputusan, pengguna pilot, dan lingkungan deployment.
2. Setujui atau ubah tahapan MVP di atas serta data profil yang boleh diproses pada pilot.
3. [Disetujui 2026-11-10] Matriks aksi/cakupan telah disetujui pemilik produk dan pemilik keamanan; dicatat di §11 dan diterapkan di `backend/cmd/server/main.go`. Pemilik organisasi masih perlu memvalidasi hierarki unit untuk assignment `homebaseId` operator/atasan.
4. Pemilik identitas/IT menyetujui provider, provisioning, sesi/token, MFA, trusted origins, dan topology frontend/API.
5. Pemilik platform/DB menyetujui tool migrasi, backup/restore, RPO/RTO, dan deployment; pemilik kebijakan menentukan retensi audit.
6. Pemilik keuangan mengidentifikasi sumber sistem/rekaman konfirmasi pembayaran yang otoritatif serta pihak yang boleh mengesahkan.
7. Pemilik platform menentukan kebutuhan Redis/queue; pemilik komunikasi/IT menentukan provider email, domain pengirim, retry, dan kebijakan isi.

Sampai keputusan dan penanggung jawab tersebut dicatat serta disetujui, Gerbang 0 pada `docs/IMPLEMENTATION-PLAN.md` tetap belum terpenuhi. Jawaban “ya” pada pertanyaan konfirmasi pembayaran dan integrasi belum menentukan rekaman sumber, provider, atau kebijakan operasional; tidak mengaktifkan pengiriman email.
