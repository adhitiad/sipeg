# Threat Model — SIMPEG

Tanggal: 2026-11-11
Tujuan: Threat modeling untuk area data pegawai, persetujuan, pembayaran, cuti,
impor/ekspor, dan kegagalan dependency sesuai **OWASP Top 10:2025 A06 Insecure
Design**. Dokumen ini mencakup ancaman, dampak, dan mitigasi yang sudah diterapkan
atau perlu ditindaklanjuti.

## 1. Data Pegawai (Employee Data)

| Ancaman | Dampak | Mitigasi yang Terdapat | Mitigasi yang Perlu Ditindaklanjuti |
|---------|--------|------------------------|-------------------------------------|
| **IDOR/BOLA** (A01) — Akses data pegawai lain melalui parameter ID yang diubah. | Pencahayaan data pribadi pegawai (NIK, NIP, alamat, gaji pokok). | Deny-by-default di setiap endpoint. Cakupan unit ditegakkan via `ScopeHomebaseIDs` dan per-record check pada use case baca/ubah/arsip. | Tidak ada — sudah diterapkan dan diuji di `scope_handlers_test.go`. |
| **Elevasi role via header** (A01) — Pengguna meniru header Authorization atau role untuk naik ke admin. | Akses ke seluruh organisasi, payroll, dan konfigurasi. | Role selalu dibaca dari tabel `Role` pada setiap resolusi sesi, bukan dari input klien. Header role dari browser diabaikan. | Tidak ada — sudah diuji di `auth_test.go` dan `gorm_resolver_test.go`. |
| **Duplikasi identitas** — NIP/NIDN/NIDK duplikat. | Data pegawai ganda, keakuratan slip gaji terganggu. | Validasi uniqueness di `CreateEmployees` dan `CreateEmployee` dengan conflict error. | Tidak ada — sudah diterapkan di `gorm_employee.go`. |
| **Soft delete tanpa audit** — Arsip pegawai tidak tercatat siapa dan kapan. | Tidak dapat menelusuri siapa yang menonaktifkan pegawai. | Setiap arsip menulis jejak audit aktor/aksi/target secara transaksional. | Tidak ada — sudah diterapkan di `gorm_employee.go`. |

## 2. Persetujuan dan Persetujuan Kerja (Approval Workflow)

| Ancaman | Dampak | Mitigasi yang Terdapat | Mitigasi yang Perlu Ditindaklanjuti |
|---------|--------|------------------------|-------------------------------------|
| **Lompat approval** — Pengguna menyetujui sendiri cuti/mutasi/pembayaran tanpa persetujuan atasan. | Persetujuan palsu, pembayaran tidak sah. | Persetujuan dibatasi ke role yang memiliki grant eksplisit (`employee:role:manage` hanya admin, `payroll:manage` hanya admin). | Aturan antrean approval berjenjang untuk cuti/mutasi belum ditetapkan (belum dibuat modul `internal/leave`). |
| **Perubahan role tidak tercatat** — Role diubah tanpa jejak. | Akses berlebihan tidak dapat ditelusuri. | Perubahan role via `PUT /api/v1/employees/{id}/role` menulis jejak `employee.role.updated` secara transaksional. | Tidak ada — sudah diterapkan. |

## 3. Pembayaran dan Slip Gaji (Payroll)

| Ancaman | Dampak | Mitigasi yang Terdapat | Mitigasi yang Perlu Ditindaklanjuti |
|---------|--------|------------------------|-------------------------------------|
| **Pembayaran duplikat** — Slip dibayar lebih dari satu kali. | Pemberitahuan email gaji dikirim duplikat, keuangan berlebih. | Slip hanya dapat dibayar sekali dan hanya dari status `TERBIT` → `TERBAYAR`. Notifikasi memakai kunci idempotensi `salary-paid:{periode}:{userId}`. | Tidak ada — sudah diterapkan di `gorm.go` dan `payroll_handlers.go`. |
| **Notifikasi tanpa bukti** — Email "gaji sudah masuk" dikirim sebelum pembayaran dikonfirmasi. | Pegawai mengira gaji sudah masuk padahal belum. | Email gaji hanya dikirim setelah baris `pembayaran` tersimpan. Konstruktor `notify.SalaryPaidNotice` tidak menerima nominal. | Tidak ada — sudah diterapkan di `notify` dan `payroll`. |
| **Nominal bocor ke email** — Slip gaji berisi nominal dalam email. | Pencahayaan data gaji sensitif melalui email. | Pesan email hanya berisi nama, periode, dan tautan — tidak pernah memuat nominal. | Tidak ada — sudah ditegakkan secara konstruktif di `notify`. |
| **Slip diakses pegawai lain** — Pegawai A melihat slip pegawai B. | Pencahayaan informasi gaji antar pegawai. | `payroll:read:self` membatasi ke slip milik sendiri. `payroll:read:homebase` dibatasi via `ScopeHomebaseIDs` dan pengecekan `HomebaseId` pemilik slip. | Tidak ada — sudah diuji di `scope_handlers_test.go`. |

## 4. Cuti dan Persetujuan Cuti (Leave)

| Ancaman | Dampak | Mitigasi yang Terdapat | Mitigasi yang Perlu Ditindaklanjuti |
|---------|--------|------------------------|-------------------------------------|
| **Cuti tumpang tindih** — Pengajuan cuti baru menutupi cuti yang belum disetujui. | Saldo cuti borong, ketidakteraturan kehadiran. | Validasi urutan tanggal (`tanggalSelesai` tidak boleh sebelum `tanggalMulai`) pada `ValidasiCuti` dan `ApplyCutiPatch`. | Pencegahan tumpang tindih periode per pegawai pada cuty disetujui belum diimplementasi di use case; tetapkan sebelum modul dipakai produksi. |
| **Persetujuan cuti oleh orang yang tidak berwenang** — Atasan menyetujui cuti pegawai di luar cakupannya. | Cuti disetujui tanpa kontrol yang tepat. | `canApprove` membatasi ke unit penugasan beserta turunan (recursive CTE Homebase) melalui `ScopeHomebaseIDs` dan pengecekan `HomebaseId` pemilik pengajuan per record; `403` bila tidak berhak. | Tidak ada — sudah diterapkan dan diuji di `scope_handlers_test.go` dan `organization_scope_test.go`. |
| **Bukti pengajuan palsu atau tidak lengkap** — Cuti sakit/melahirkan/harian disetujui tanpa eviden yang mencukupi. | Pengajuan cuti disetujui tanpa dasar yang dapat ditanggungjawabkan. | Eviden wajib saat pengajuan dibuat (`422` bila kosong) dan dicek ulang saat approve; jenis eviden terstruktur (`SURAT_DOKTER`/`FOTO`/`LAINNYA`) dengan field wajib per jenis; cuti sakit/melahirkan hanya boleh `SURAT_DOKTER`, harian hanya `FOTO`/`LAINNYA`. | Backend tidak memverifikasi isi berkas bukti eksternal (URL dan klaim timestamp diambil apa adanya); approver harus memeriksa berkas secara visual. Pertimbangkan verifikasi berkas pada waktu unggah jika penyimpanan bukti dibuat internal. |

## 5. Impor dan Ekspor (Import/Export)

| Ancaman | Dampak | Mitigasi yang Terdapat | Mitigasi yang Perlu Ditindaklanjuti |
|---------|--------|------------------------|-------------------------------------|
| **Impor massal tanpa audit** — Impor 1000 pegawai tanpa mencetak siapa yang mengimpor. | Tidak dapat menelusuri sumber data palsu. | Impor massal menulis jejak `employee.imported` dengan jumlah dan unit cakupan. | Tidak ada — sudah diterapkan di `gorm_employee.go`. |
| **Ekspor data sensitif** — Ekspor seluruh data pegawai termasuk NIK dan gaji. | Pencahayaan massal data pribadi. | Belum ada fitur ekspor. | Jika fitur ekspor dibuat, wajib diberi permission `employee:export` (admin-only), diaudit, dan tidak boleh menyertakan field sensitif (NIK, NPWP, rekening bank) kecuali diperlukan. |

## 6. Kegagalan Dependency (External Service Failure)

| Ancaman | Dampak | Mitigasi yang Terdapat | Mitigasi yang Perlu Ditindaklanjuti |
|---------|--------|------------------------|-------------------------------------|
| **Database down** — PostgreSQL tidak tersedia. | API gagal total, tidak dapat login atau mengakses data. | Readiness check (`/ready`) mengembalikan 503 bila DB tidak terjangkau. Timeout statement diterapkan via `DB_STATEMENT_TIMEOUT`. | Tidak ada — sudah diterapkan. |
| **Redis down** — Cache gagal. | Pembacaan kategori pegawai lebih lambat (fallback ke PostgreSQL). | Cache-saja dengan `ReadThrough` — gagal Redis tidak membaca dari cache, langsung ke PostgreSQL. | Tidak ada — sudah diterapkan di `internal/cache`. |
| **QStash down** — Email notifikasi tidak terkirim. | Notifikasi slip gaji tertunda, outbox menumpuk. | Outbox persisten di PostgreSQL memastikan tidak ada pesan hilang. QStash retry dengan backoff, dead-letter setelah 3 percobaan. | Tidak ada — sudah diterapkan di `internal/outbox` dan `internal/qstash`. |
| **SMTP down** — Email tidak terkirim. | Pegawai tidak menerima notifikasi slip. | Outbox menyimpan pesan, worker mencoba ulang. Kesalahan permanen menjadi dead-letter untuk ditinjau manual. | Tidak ada — sudah diterapkan di `internal/outbox` dan `internal/notify`. |
| **Penyedia hari libur down** — api.co.id tidak tersedia. | Kalender hari libur tidak diperbarui, tetapi pembacaan tetap dari tabel lokal. | Pembacaan kalender selalu dari tabel `HariLibur` lokal, tidak bergantung pada penyedia. Penyegaran hanya dipicu admin bila kunci tersedia. | Tidak ada — sudah diterapkan di `internal/holiday`. |

## 7. Keamanan Tambahan (Sudah Diterapkan)

| Area | Mitigasi | Bukti |
|------|----------|-------|
| **RLS default-deny** | Seluruh 45+ tabel aplikasi memiliki RLS aktif tanpa FORCE. | `backend/migrations/000004_row_level_security.sql` |
| **GORM-only** | Supabase Data API tidak diekspos. Backend adalah satu-satunya jalur ke database. | `DEC-019`, `internal/config/config.go` |
| **Rate limit** | Batas laju per klien, lebih ketat untuk endpoint autentikasi. | `internal/httpserver/ratelimit.go` |
| **Error uniform** | Bentuk error serupa tanpa bocor detail internal. | `internal/apierror`, `internal/httpserver/errors.go` |
| **Audit append-only** | Tidak ada jalur hapus/ubah pada `auditLog` dari kode aplikasi. | `internal/audit/audit.go` + tes pemindai source |
| **Penolakan tercatat** | Penolakan di gerbang izin menjadi entri audit `authz.denied`; seluruh respons 401, 403, dan 429 menjadi security event terstruktur berisi pelaku, IP, dan request ID. | `internal/auth/denial.go`, `internal/observability/security.go` |
| **Image container tanpa gosu** | Image runtime `gcr.io/distroless/static-debian12:nonroot` hanya memuat 3 binary statis; `usr/sbin` kosong, tidak ada shell, tidak ada `gosu`. Go dinaikkan ke `1.27.4` (tambalan `GO-2026-6859` / CVE-2026-97054 pada `net/http`); `govulncheck` bersih. | `backend/Dockerfile`, `DEC-045`, `docker export` verifikasi isi image |

## 8. Ancaman yang Perlu Ditindaklanjuti (Backlog)

1. **Threat modeling untuk fitur yang belum dibuat:** impor/ekspor dan arsip jejak audit perlu threat modeling terpisah saat modul dibuat. Modul cuti sudah diimplementasi dan dimodelkan pada bagian 4.
2. **Uji penetrasi terhadap endpoint admin dan ekspor** — belum ada fitur ini, sehingga belum dapat diuji.
3. **Uji timeout terhadap provider eksternal nyata** — membutuhkan kredensial staging QStash/SMTP.
