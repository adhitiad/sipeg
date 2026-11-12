# Backend

Backend SIMPEG menggunakan Go 1.27.4 (lihat `backend/go.mod`), GORM, dan PostgreSQL. `internal/models` memetakan tabel aplikasi yang dibuat oleh migrasi Goose di `migrations/`; migrasi SQL adalah sumber skema yang ditinjau dan diterapkan.

Gunakan `internal/database.OpenPostgres` agar nama tabel dan kolom mengikuti konvensi PostgreSQL yang dipakai migrasi. Model menyimpan foreign key sebagai kolom UUID; relasi untuk kebutuhan query dapat ditambahkan bersama fitur yang memakainya.

Kontrak HTTP/JSON MVP ditulis dalam OpenAPI 3.1 di `../docs/openapi.yaml`. Seluruh endpoint data berada di bawah prefix versi `/api/v1`; endpoint operasional (`/health`, `/ready`) tidak berversi. Grant runtime ada di `cmd/server/main.go`; status implementasi cakupan akses tidak boleh disimpulkan hanya dari kontrak endpoint.

## Versi API

Versi API memakai prefix path: `/api/v1/employees`, `/api/v1/organizations`, dan seterusnya. Konstanta `httpserver.APIPrefix` dan `httpserver.Version` dipakai di route dan tes — jangan menulis path `/api/...` secara literal.

- Path `/api/...` tanpa versi ditolak `400 unversioned_path` dengan pesan yang memuat prefix yang benar.
- Path dengan versi yang tidak dikenal, misalnya `/api/v2/employees` saat ini versi `v1`, menghasilkan `404 not_found`.
- Setiap respons memuat header `X-API-Version`.
- Versi naik hanya bila ada perubahan yang merusak kompatibilitas. Perubahan additive tetap pada `v1`; versi lama tetap didukung selama masa transisi.

Endpoint operasional `/health` dan `/ready` berada di luar prefix versi.

Atur `DATABASE_URL` pada aplikasi yang memanggil backend dan berikan nilainya ke `OpenPostgres`. Migration runner yang dipilih adalah Goose dengan konfigurasi di `.goose.yaml`; workflow deployment tersedia di `.github/workflows/deploy-migrations.yml` (dipicu secara manual via `workflow_dispatch`, `DATABASE_URL` sebagai secret GitHub, belum pernah dijalankan di GitHub karena environment `staging`/`production` dan secret belum dibuat; urutan perintah diuji manual terhadap DB uji). Migrasi telah diterapkan pada database produksi `postgres` dan database uji `simpeg_perf` hingga versi Goose 19. Tinjau dan terapkan melalui proses migrasi yang disetujui sebelum memakai CRUD. Jangan menjalankan `AutoMigrate`.

## Struktur lapisan

Backend dibagi per area bisnis, dan tiap area memisahkan empat lapis:

| Lapis | Isi | Aturan |
| --- | --- | --- |
| `domain` | Aturan, validasi, dan nilai tanpa I/O. | Tidak mengimpor HTTP maupun database. |
| `app` | Use case: memeriksa permission, membatasi cakupan unit, lalu memanggil port repository. | Tidak menulis SQL dan tidak menulis respons HTTP. |
| `repository` | Port persistensi dan adapter GORM. | Satu-satunya lapisan yang menulis SQL. |
| `httpserver` | Transport: dekode permintaan, panggil use case, petakan error ke status HTTP. | Tidak memuat aturan bisnis. |

Area yang sudah dipisah: `internal/directory/{domain,app,repository}` (pegawai, unit organisasi, kategori, jejak audit), `internal/payroll/{domain,app,repository}` (slip gaji, konfirmasi pembayaran, jadwal gajian per unit), dan `internal/holiday` (kalender hari libur nasional). Kontrak daftar bersama ada di `internal/listing` (`listing.Options`, `listing.Page`).

## Mata uang dan zona waktu

Mata uang sistem adalah Rupiah Indonesia (IDR) dan tidak ada konversi antar-mata uang. Nominal diformat hanya lewat `internal/money`: `money.Format(1500000)` menghasilkan `Rp 1.500.000` (pemisah ribuan titik, desimal koma).

Zona waktu operasional adalah Asia/Jakarta (UTC+7, WIB). Penyimpanan timestamp memakai UTC: `database.OpenPostgres` memaksa parameter runtime `timezone=UTC` pada DSN sehingga default `now()` dan nilai yang ditulis GORM memakai basis yang sama. Konversi ke waktu lokal dilakukan saat tampilan dan aturan kalender melalui `internal/timeutil` (`timeutil.InLocation`, `timeutil.FormatDate`). Kolom tanggal pada skema acuan (`date`) menyimpan kalender tanpa jam; nilainya ditulis sebagai tengah malam UTC agar tanggal yang dibaca kembali tetap sama setelah dikonversi.

## Konfigurasi runtime

| Variabel | Kegunaan |
| --- | --- |
| `DATABASE_URL` | Wajib. Connection string PostgreSQL. |
| `APP_ENV` | `development` (default) atau `production`. |
| `PORT` | Default `8080`. |
| `BETTER_AUTH_SECRET` | Shared secret verifikasi JWT Better Auth. Wajib pada production, minimal 32 karakter, dan tidak boleh memuat pola bawaan. |
| `TRUSTED_ORIGINS` | Origin browser yang diizinkan memanggil API (pisahkan dengan koma). Wildcard `*` ditolak. Kosong berarti tidak ada akses lintas origin. Wajib pada production. |
| `DB_MAX_OPEN_CONNS` | Batas koneksi terbuka, default `10`. Pooler sesi Supabase membatasi jumlah koneksi concurrently per proyek; bila batas ini melampaui batas pooler, PostgreSQL menolak koneksi baru (`EMAXCONNSESSION`) dan backend menjawab `503`. |
| `DB_MAX_IDLE_CONNS` | Batas koneksi idle, default `5`, tidak boleh melebihi `DB_MAX_OPEN_CONNS`. |
| `DB_CONN_MAX_LIFETIME` | Umur maksimum koneksi, default `30m`. |
| `DB_STATEMENT_TIMEOUT` | Batas waktu satu statement, default `15s`. |
| `RATE_LIMIT_REQUESTS` | Batas permintaan per klien per jendela waktu, default `120`. |
| `RATE_LIMIT_WINDOW` | Panjang jendela rate limit, default `1m`. |
| `RATE_LIMIT_AUTH_REQUESTS` | Batas lebih ketat untuk `/api/v1/auth` dan `/api/v1/me`, default `10`. |
| `RATE_LIMIT_AUTH_WINDOW` | Panjang jendela rate limit autentikasi, default `1m`. |
| `APP_BASE_URL` | Origin aplikasi untuk tautan pada email, misalnya `https://simpeg.unisubang.ac.id`. |
| `AUDIT_RETENTION_DAYS` | Masa simpan jejak audit dalam hari, default `90`. `0` berarti tanpa pemotongan otomatis. |
| `RPO_HOURS` | Target pemulihan data dalam jam, default `24`. |
| `RTO_HOURS` | Target pemulihan layanan dalam jam, default `4`. |
| `SLO_TARGET` | Target ketersediaan layanan dalam persen, default `99.9`. |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_SECURE`, `SMTP_FROM` | Konfigurasi SMTP. `SMTP_FROM` wajib diisi bila `SMTP_HOST` diisi dan harus berupa alamat email yang sah; bentuk baku `"Nama Tampilan" <alamat@domain>` juga diterima. `SMTP_SECURE=true` memakai TLS langsung (porta 465), `false` memakai koneksi polos yang dinaikkan lewat STARTTLS (porta 587), dan kosong berarti mode ditebak dari porta. |
| `QSTASH_URL`, `QSTASH_SCHEDULES_URL`, `QSTASH_TOKEN`, `QSTASH_CURRENT_SIGNING_KEY`, `QSTASH_NEXT_SIGNING_KEY` | QStash untuk penjadwalan pekerjaan asinkron. `QSTASH_CURRENT_SIGNING_KEY` wajib diisi bila `QSTASH_TOKEN` diisi. `QSTASH_URL` boleh diisi root API regional; jalur `/v2/publish` dan `/v2/schedules` ditambahkan otomatis. `QSTASH_SCHEDULES_URL` boleh dikosongkan karena endpoint jadwal diturunkan dari `QSTASH_URL`. |
| `RABBITMQ_URL` | Broker RabbitMQ untuk pekerjaan terjadwal self-hosted (docker/podman). Bila diisi, server menjalankan konsumen pekerjaan dan melewati pendaftaran jadwal QStash (`DEC-042`). Penjadwal `cmd/jobsched` membaca variabel yang sama; `SLO_EVALUATE_ENABLED` penjadwal harus sama dengan `SLO_ENABLED` backend. |
| `UPSTASH_REDIS_REST_URL`, `UPSTASH_REDIS_REST_TOKEN` | Cache Redis Upstash untuk data referensi non-sensitif saja. |
| `HOLIDAY_API_URL`, `HOLIDAY_API_KEY` | Penyedia kalender hari libur nasional (api.co.id). Hanya dipakai untuk penyegaran yang dipicu admin; pembacaan kalender selalu dari tabel `HariLibur`. Bila `HOLIDAY_API_KEY` kosong, penyegaran menolak (`503`) dan kalender bersifat baca saja. |
| `SLO_ENABLED`, `SLO_WINDOW_MINUTES`, `SLO_P95_TARGET_MS`, `SLO_ERROR_RATE_TARGET`, `SLO_MIN_REQUESTS`, `SLO_ALERT_RECIPIENTS`, `SLO_RETENTION_DAYS` | Pemeriksaan target layanan terjadwal. `SLO_ENABLED=false` (default) berarti pengoleki metrik tidak berjalan sama sekali. Bila diaktifkan, backend menulis metrik ke tabel `sloMetric` dan mendaftarkan pemeriksaan berkala `slo.evaluate` melalui QStash. |

Pada `APP_ENV=production` backend menolak start bila secret lemah atau `TRUSTED_ORIGINS` kosong — konfigurasi tidak aman tidak boleh lolos diam-diam.

Batas laju diimplementasikan in-memory per instance. Pada deployment multi-instance, batas efektif menjadi jumlah instance dikalikan batas per instance; batas terdistribusi membutuhkan penyimpanan bersama (misalnya Redis) dan itu bukan use case Redis yang disetujui (`DEC-015` hanya menyetujui cache data referensi).

Tindakan operator saat insiden dijelaskan di `../docs/RUNBOOK.md`.

## Autentikasi dan otorisasi

Endpoint `GET /api/v1/me` memerlukan `Authorization: Bearer <session-token>`. Backend mendukung dua jenis sesi: JWT Better Auth (diverifikasi signature HMAC dengan `BETTER_AUTH_SECRET`) dan sesi opaque pada tabel `session` yang ditolak bila kedaluwarsa; keduanya menghasilkan principal dari data server-side (identitas dan role). Token yang tidak valid menghasilkan `401`; endpoint tanpa autentikasi tidak memperoleh principal.

Middleware `RequirePermission` hanya mengizinkan grant eksplisit untuk nama role dan permission; grant yang tidak didefinisikan ditolak. `RequireAnyPermission` mengizinkan salah satu dari beberapa permission (dipakai pada route daftar/detail pegawai) dan tetap menolak bila tidak ada yang cocok. Matriks role disetujui dan dikonfigurasi secara statis di `cmd/server/main.go` (rinciannya ada di `../docs/PRD.md` §11). `auth.CanAccessUserInScope` menerapkan pemeriksaan self/homebase pada use case baca, ubah, dan arsip pegawai di lapisan aplikasi.

Belum ada login, OAuth callback, logout, atau penerbitan sesi di backend; endpoint `/api/v1/me` hanya memverifikasi sesi yang sudah ada. Endpoint `POST /api/v1/auth/logout` tersedia dan menghapus baris `session` sehingga token tidak lagi berlaku; logout bersifat idempoten. Endpoint ini hanya mencabut sesi opaque di database — penrevokasian token JWT milik Better Auth tetap dilakukan oleh runtime autentikasi. Provider autentikasi (Better Auth) dan matriks role sudah diputuskan, tetapi detail berikut masih harus ditetapkan dan diuji sebelum rilis: bagaimana frontend mengirim token/sesi, rotasi/revokasi sesi, CSRF/cookie, provisioning akun, dan MFA. Akun nonaktif dan sesi invalid ditolak. Jangan meneruskan `RoleName` atau identitas dari header selain sesi terverifikasi.

## Health, readiness, dan observability

`GET /health` adalah liveness: selalu `200` selama proses hidup, dan tidak gagal hanya karena dependency eksternal sedang tidak tersedia. `GET /ready` adalah readiness: memeriksa dependency wajib (saat ini ping PostgreSQL) dengan timeout dan mengembalikan `503` bila belum siap. Detail kegagalan tidak dikembalikan agar tidak membocorkan host, connection string, atau kredensial. Jangan menggabungkan kedua endpoint tersebut.

Log ditulis sebagai JSON satu baris per entri oleh paket `internal/observability`, dengan correlation ID pada header `X-Request-ID`. Request ID dari klien dipakai hanya bila karakternya aman; selain itu digenerate ulang. Header kredensial, body request, dan nilai query string tidak pernah dicatat. Log ditulis ke stdout, jadi pengelolaan dan retensinya mengikuti kebijakan deployment.

## Keamanan endpoint

`httpserver.CORSMiddleware` hanya memberi header CORS untuk origin yang tercantum di `TRUSTED_ORIGINS`; origin lain tidak mendapat akses lintas origin, dan preflight dari origin tak dikenal ditolak `403`. `httpserver.SecurityHeadersMiddleware` menambahkan `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Cross-Origin-Opener-Policy`, dan `Permissions-Policy` pada setiap respons. `httpserver.RateLimiter` membatasi permintaan per klien dengan batas lebih ketat pada endpoint autentikasi. `httpserver.StatementTimeoutMiddleware` membungkus context permintaan dengan batas waktu statement database.

## Integrasi layanan pendukung

Keempat integrasi diimplementasikan sebagai paket terpisah di `internal/`.

**Outbox persisten** (`internal/outbox`) adalah fondasi. Transaksi bisnis menulis baris outbox dalam transaksi yang sama sehingga pekerjaan tidak hilang saat queue eksternal gagal. Baris diklaim memakai `FOR UPDATE SKIP LOCKED`, kunci `dedupeKey` mencegah pengiriman ganda, retry memakai backoff eksponensial, dan pesan yang melewati batas percobaan diletakkan pada status `FAILED` sebagai dead-letter. Meja `outboxMessage` dibuat lewat migrasi `000006_outbox_message.sql`.

**QStash** (`internal/qstash`) adalah penjadwal pekerjaan asinkron. Callback diverifikasi signature HS256 terhadap **raw request body**, URL tujuan, dan masa berlaku, dengan dukungan rotasi kunci (kunci saat ini dan berikutnya). Endpoint callback yang terdaftar: `POST /api/v1/internal/qstash/audit/cleanup` — ia menjalankan retensi jejak audit dan dicatat sebagai aksi `audit.cleaned`; dan `POST /api/v1/internal/qstash/payroll/notify` — ia menjalankan pemberitahuan ketersediaan slip gaji H+1 dan dicatat sebagai aksi `payroll.notified`. Keduanya **tidak** memakai sesi bearer; pengesahnya signature QStash. Jadwal harian didaftarkan lewat `internal/jobs.EnsureSchedule` yang idempoten: jadwal dengan tujuan yang sama tidak didaftarkan dua kali. Jadwal pemberitahuan H+1 didaftarkan pada `5 19 * * *` (02:05 WIB) dengan 3 percobaan ulang.

**RabbitMQ** (`internal/mq`) adalah pengganti penjadwal QStash pada deployment self-hosted (docker/podman) — `DEC-042`. Bila `RABBITMQ_URL` diisi, server menjalankan konsumen pekerjaan (`mq.Consumer`) dan melewati pendaftaran jadwal QStash; payload dan kind sama dengan callback QStash (`{"kind": "audit.cleanup"}` dan seterusnya), sehingga kedua transport menjalankan logika pekerjaan yang identik. Penjadwal `cmd/jobsched` menerbitkan pesan dari tick cron UTC yang sama dengan jadwal QStash, dengan publisher confirm dan pesan persistent. Pesan yang gagal dijalankan pindah ke antrean `simpeg.jobs.dead` (dead-letter); tick berikutnya memicu eksekusi baru. Topologi (exchange, antrean, binding) dideklarasikan idempoten saat startup. Detail operasional ada di `rabbitmq/README.md` dan `docs/RUNBOOK.md` §8B.

**Migrasi** (`cmd/migrate`) membawa seluruh migrasi Goose dalam binary (embedded filesystem dari `migrations/migrations.go`) sehingga container migrasi tidak memerlukan CLI terpisah. `migrate` menerapkan migrasi yang belum jalan dan `migrate status` menampilkan statusnya; keduanya idempoten dan aman diulang.

**Email** (`internal/notify`) memakai SMTP. `notify.SalaryPaidNotice` tidak menerima parameter nominal — nominal gaji hanya tersedia di halaman terautentikasi dan dirujuk lewat tautan. `notify.EmailDeliverer` menjembatani antrean outbox ke SMTP: worker background mengirim pesan yang diklaim dispatcher. Kesalahan permanen (payload rusak, jenis tidak dikenal, sumber konfirmasi pembayaran belum dikonfigurasi) langsung menjadi dead-letter tanpa diulang; kegagalan sementara dijadwalkan ulang dengan backoff.

Alamat pengirim bawaan adalah `no-reply@simpeg.unisubang.ac.id` dengan origin aplikasi `https://simpeg.unisubang.ac.id`. Backend memvalidasi bentuk alamat pengirim saat start dan menolak alamat yang tidak sah.

**Sumber konfirmasi pembayaran sudah ada.** Modul penggajian menyimpan bukti pembayaran pada tabel `pembayaran` yang menempel pada slip gaji, dan slip ditandai `TERBAYAR`. `notify.EmailDeliverer.Confirmations` memakai `payroll/app.Service.ConfirmedPayment` sehingga notifikasi gaji hanya terkirim bila pembayaran benar-benar tercatat. Notifikasi gaji tetap tidak pernah memuat nominal.

**Redis** (`internal/cache`) hanya untuk cache data referensi. `ReadThrough` selalu memuat ulang dari sumber utama bila cache miss atau Redis gagal, sehingga cache tidak pernah menjadi sumber kebenaran. Kunci yang menyerupai sesi, token, kredensial, penggajian, atau status pembayaran ditolak saat runtime. Cache daftar kategori pegawai disambungkan ke `GET /api/v1/categories`; kalender hari libur disambungkan ke `GET /api/v1/holidays?year=` (`holiday:{year}`, TTL 24 jam). Tanpa Redis konfigurasi pembacaan tetap langsung ke PostgreSQL.

Penyimpanan objek S3-compatible belum masuk scope dan mekanismenya sengaja tidak dibuat.

## Format error

Semua endpoint memakai bentuk respons yang sama melalui paket `internal/apierror`:

```json
{"error": {"code": "...", "message": "...", "requestId": "...", "fields": {...}, "rows": [...]}}
```

`code` selalu ada; `message`, `requestId`, `fields`, dan `rows` bersifat opsional sesuai jenis error. Gunakan konstanta `apierror.Code*` dan jangan menulis bentuk error sendiri di paket lain. Respons `404` dan `405` yang dihasilkan router juga dikonversi ke bentuk JSON yang sama oleh `httpserver.JSONErrorMiddleware`.

## Direktori pegawai dan organisasi

CRUD pegawai tersedia di `/api/v1/employees`; organisasi tersedia di `/api/v1/organizations` untuk unit, fakultas, jurusan, biro, dan bagian. Semua jenis menggunakan satu pohon induk-anak fleksibel. `DELETE` mengarsipkan record, tidak menghapus riwayat; organisasi dengan pegawai aktif atau anak aktif tidak dapat diarsipkan. Perubahan direktori dicatat ke `AuditLog` tanpa menyalin nilai profil ke log.

Permission endpoint pegawai adalah `employee:read:any`, `employee:read:homebase`, `employee:read:self`, `employee:create`, `employee:create:homebase`, `employee:update:any`, `employee:update:homebase`, `employee:archive`, `employee:import`, dan `employee:import:homebase`; organisasi memakai `organization:read`, `organization:read:homebase`, `organization:manage`, dan `organization:manage:homebase`; jejak audit memakai `audit:read` dan `audit:read:homebase`; penggajian memakai `payroll:read:any`, `payroll:read:homebase`, `payroll:read:self`, `payroll:manage`, `payroll:schedule`, dan `payroll:schedule:homebase`; kalender hari libur memakai `holiday:read` dan `holiday:manage`. Grant role dikonfigurasi di `cmd/server/main.go` sesuai matriks di `docs/PRD.md` §11.

Permission berakhiran `:any` berarti akses lintas seluruh organisasi dan hanya diberikan kepada `admin`. Permission berakhiran `:homebase` berarti akses dibatasi pada unit penugasan principal beserta seluruh turunannya pada pohon `Homebase`; `operator` dan `atasan` memakai permission ini. Permission berakhiran `:self` hanya memberi akses ke record milik principal sendiri. Cakupan ditegakkan di lapisan aplikasi: daftar difilter dengan `listing.Options.ScopeHomebaseIDs`, setiap record diperiksa per-pegawai pada use case baca, ubah, dan arsip, unit organisasi diperiksa per-unit, slip gaji diperiksa terhadap unit pegawai pemiliknya, dan pegawai hasil impor operator ditempatkan pada unit penugasannya (template impor tidak memuat kolom unit). Operator hanya dapat membuat unit baru di bawah induk yang sudah berada dalam cakupan; unit akar hanya dapat dibuat oleh `admin`. Principal tanpa penugasan unit tidak melihat data pegawai, unit, slip gaji, maupun jejak audit apa pun.

**Batas yang belum terpenuhi:** jejak audit untuk perubahan role belum ada karena endpoint perubahan role belum dibuat. Ingestion presensi hanya diberikan kepada `admin` dan belum memakai autentikasi perangkat.

## Modul penggajian

Slip gaji tersedia di `/api/v1/payroll/slips`. `GET` daftar dan detail slip, `POST /api/v1/payroll/slips/{payrollId}/payment` mencatat konfirmasi pembayaran (hanya `admin`).

Konfirmasi pembayaran dicatat pada tabel `pembayaran` yang menempel pada slip gaji, lalu slip ditandai `TERBAYAR` beserta `tanggalDibayar`, `konfirmasiPembayaranAt`, dan `konfirmasiOlehId`. Status slip pada skema acuan adalah `DRAF`, `TERBIT`, dan `TERBAYAR`; slip hanya dapat dibayar satu kali dan hanya dari status `TERBIT`.

Pencatatan pembayaran, antrean notifikasi gaji, dan jejak audit `payroll.payment.confirmed` ditulis dalam satu transaksi. Karena itu notifikasi gaji tidak pernah ada tanpa bukti pembayaran yang tersimpan, dan pengiriman ganda dicegah oleh kunci idempotensi outbox.

Konfirmasi pembayaran pada MVP dimasukkan manual oleh admin: admin mengisi referensi transaksi/rekapan bank, waktu, dan pengesah. Integrasi otomatis ke bank (REST dan websocket) disiapkan sebagai adapter lanjutan yang menulis bukti pembayaran ke tabel `pembayaran` lewat jalur transaksional yang sama; implementasinya menunggu kontrak API vendor dan tidak dibuat tanpa pemakaian.

### Jadwal gajian dan pemberitahuan slip H+1

Tanggal gajian bersifat tetap (1–28) per unit organisasi, diinput lewat `GET/PUT /api/v1/payroll/schedules` — `admin` untuk seluruh unit (`payroll:schedule`), `operator` hanya unit dalam cakupan penugasannya beserta turunannya (`payroll:schedule:homebase`). Nilai disimpan di tabel `jadwalGaji` (satu baris per unit).

Slip berstatus `TERBIT` atau `TERBAYAR` pada unit yang punya jadwal gajian diberi pemberitahuan **satu hari kalender setelah** tanggal gajian. Tanggal H+1 dipakai tetap — tidak digeser karena jatuh pada Sabtu, Minggu, atau hari libur nasional. Pemberitahuan adalah pemberitahuan **ketersediaan slip**, bukan bukti pembayaran: isi pesan hanya nama, periode, dan tautan — tidak pernah memuat nominal gaji dan tidak menyatakan gaji sudah dibayar. Pesan disimpan di outbox sebagai jenis `email.slip-tersedia` dengan kunci idempotensi `slip-available:{slipId}`, sehingga setiap slip hanya diberi satu pemberitahuan. Pekerjaan dijalankan oleh job QStash `payroll.notify` (`5 19 * * *` = 02:05 WIB, 3 percobaan ulang) dan tercatat sebagai aksi `payroll.notified`. Notifikasi gaji ("gaji sudah masuk") tetap terpisah dan hanya boleh terkirim setelah konfirmasi pembayaran tersimpan.

### Kalender hari libur nasional

Kalender diisi dari penyedia api.co.id (`HOLIDAY_API_URL`, `HOLIDAY_API_KEY`; header `x-api-co-id`) dan disimpan di tabel `HariLibur`. `POST /api/v1/holidays/{year}/refresh` (permission `holiday:manage`, hanya admin) mengganti seluruh entri tahun yang diminta dalam satu transaksi bersama jejak audit `holiday.refreshed`; `GET /api/v1/holidays?year=` (permission `holiday:read`) selalu membaca tabel lokal dan tidak pernah bergantung pada penyedia. Entri disimpan untuk `type` `Public Holiday`, `National Holiday`, dan `Joint Holiday`; `Observance` dilewati. Kalender **tidak** mengubah tanggal gajian maupun pemberitahuan H+1. Bila `HOLIDAY_API_KEY` kosong, penyegaran menolak (`503`) dan kalender bersifat baca saja.

Seluruh nominal pada API ditulis sebagai Rupiah yang sudah diformat (`"Rp 4.250.000"`), dan periode ditulis sebagai `YYYY-MM` beserta label Bahasa Indonesianya.

## Jejak audit

Jejak audit ditulis lewat paket `internal/audit` dan bersifat **append-only** dari sisi kode aplikasi: backend tidak memiliki jalur hapus atau ubah pada entri yang sudah tertulis, dan hal itu dijaga tes yang memindai source backend. Satu-satunya penghapusan adalah retensi yang disetujui: `audit.Cleaner` menghapus entri yang lebih lama dari `AUDIT_RETENTION_DAYS` hari (default 90; `0` berarti tanpa pemotongan). Pekerjaan itu dijalankan terjadwal oleh QStash lewat callback `POST /api/v1/internal/qstash/audit/cleanup` dan tercatat sebagai aksi `audit.cleaned`. Setiap entri memuat aktor, aksi, sasaran, keterangan minimal, unit cakupan (`unitId`), alamat IP, dan waktu — isi profil pegawai tidak pernah disalin.

Akses dibatasi per unit: `admin` membaca seluruh jejak, `operator` hanya membaca jejak unit penugasannya beserta turunannya. Setiap pembacaan yang mengembalikan entri dicatat kembali sebagai aksi `audit.accessed`; pembacaan kosong tidak dicatat agar tabel tidak membengkak.

Kolom `unitId` adalah perubahan skema acuan dan harus selalu diperbarui bersama model GORM dan migrasi (`backend/migrations/000003_audit_log_unit.sql`). Retensi sudah ditetapkan; arsip dan ekspor jejak belum ada.

Schema memerlukan migration SQL [000001_directory_crud.sql](migrations/000001_directory_crud.sql). Migration runner yang dipilih adalah Goose dengan konfigurasi di `.goose.yaml` (direktori `./migrations`, `DATABASE_URL` sebagai env required). Workflow deployment tersedia di `.github/workflows/deploy-migrations.yml` dan diperkuat (`DEC-046`), tetapi belum pernah dijalankan di GitHub. Migrasi telah diterapkan dan diuji terhadap database nyata (`postgres` dan `simpeg_perf`) hingga versi Goose 19. Jangan gunakan `AutoMigrate`; tinjau dan terapkan migration secara terkendali sebelum menjalankan versi backend ini.

## Daftar, filter, dan pengurutan

Seluruh endpoint daftar memakai parameter yang sama: `page` (default 1), `pageSize` (default 20, maksimum 100), `search`, `sort`, dan `order` (`asc`/`desc`). Nilai `sort` di luar allowlist resource ditolak dengan `400 invalid_query`. Kolom SQL ditentukan server melalui allowlist di `internal/directory/domain/list.go` dan `internal/payroll/domain/sort.go`; nilai mentah dari pengguna tidak pernah masuk ke klausa `ORDER`. Nama kolom camelCase harus ditulis dengan kutip ganda (`"createdAt"`) agar tidak difold menjadi huruf kecil oleh PostgreSQL. Kontrak parameter bersama ada di `internal/listing`.

Endpoint daftar yang tersedia: `GET /api/v1/employees`, `GET /api/v1/organizations`, `GET /api/v1/audit-logs` (permission `audit:read`), `GET /api/v1/audit-logs/export` (CSV streaming, permission `audit:read`), `GET /api/v1/cuti` + `POST /api/v1/cuti` + `GET/PATCH /api/v1/cuti/{id}` + `POST /api/v1/cuti/{id}/approve,reject` (permission `cuti:read`/`cuti:create`/`cuti:update`/`cuti:approve`), `GET /api/v1/categories` (permission `category:read`), `GET /api/v1/payroll/slips` (permission `payroll:read:any`/`payroll:read:homebase`/`payroll:read:self`), dan `GET /api/v1/holidays` (permission `holiday:read`). Daftar kategori hanya mengembalikan kategori aktif. DTO audit memuat aktor, aksi, sasaran, keterangan minimal, dan waktu — isi profil pegawai tidak pernah disalin ke audit.

Eviden pengajuan cuti: `BERSALIN`, `SAKIT`, `SAKIT_BERKEPANJANGAN`, dan `IZIN_PENTING` wajib menyertakan eviden saat pengajuan dibuat dan dicek ulang saat approve. Bentuk eviden: `SURAT_DOKTER` (berkas + nomor + tanggal surat dokter, untuk sakit/melahirkan), `FOTO` (berkas + waktu tangkap), atau `LAINNYA` (waktu + keterangan, untuk harian). Kolom model: `jenisEviden`, `fileEvidenUrl`, `waktuEviden`, `catatanEviden` (kolom lama `fileSuratUrl` di-rename menjadi `fileEvidenUrl` lewat migrasi `000019_cuti_eviden.sql`).

Manajemen dokumen pegawai/dosen: endpoint `GET/POST/PATCH/DELETE /api/v1/arsip{...}` (permission `arsip:read`/`arsip:create`/`arsip:update`/`arsip:archive`). Berkas diunggah langsung ke Cloudinary oleh klien; backend hanya menyimpan URL dan metadata berkas (`fileUrl`, `fileSize`, `mimeType`, `cloudinaryPublicId`). Setiap jenis dokumen memiliki aturan format MIME dan ukuran maksimal (lihat `docs/PRD.md` dan `docs/DECISIONS.md` DEC-047). Konfigurasi Cloudinary: `CLOUDINARY_CLOUD_NAME`, `CLOUDINARY_API_KEY`, `CLOUDINARY_API_SECRET` (`.env.example`). Nomor rekening (`BankRekening`) dan nama rekening (`BankAtasNama`) sudah tersedia pada tabel `user`. Timestamps `createdAt`/`updatedAt` disimpan UTC dan ditampilkan Asia/Jakarta. Migrasi `000020_arsip_dokumen_tambahan.sql` menambah jenis dokumen dan kolom metadata.

## Event presensi mesin

`POST /api/v1/attendance/events` menyimpan fakta tap ternormalisasi (`machineId`, PIN mesin, waktu RFC 3339 ber-offset, dan `idempotencyKey`). Waktu disimpan sebagai UTC dengan presisi milidetik. Retry identik mengembalikan event yang sama; key yang sama dengan PIN/waktu berbeda menghasilkan `409`. PIN hanya kunci pengguna eksternal, bukan relasi langsung ke `User`; endpoint tidak membuat status harian atau menerima payload vendor/biometrik, GPS, maupun foto.

Endpoint memakai sesi bearer yang sama dengan API lain dan permission `attendance:event:ingest`; grant ini saat ini hanya diberikan kepada role `admin`. Ini mengautentikasi pemohon API, bukan perangkat. Autentikasi/kredensial perangkat dan adapter Fingerspot Revo W-230N belum tersedia karena kontrak vendor belum ditemukan. Draft [000002_attendance_events.sql](migrations/000002_attendance_events.sql) belum diterapkan; zona waktu sumber perangkat, jaminan idempotency key, pemetaan PIN, dan retensi masih perlu diputuskan. Migration belum diterapkan atau diuji terhadap database nyata.

## Performa dan uji beban

Benchmark, uji beban burst, dan uji soak memakai repository palsu sehingga tidak memerlukan database:

```bash
go test -run "^$" -bench "Benchmark" -benchmem ./internal/httpserver/... ./internal/directory/app/...
go test ./internal/httpserver/... -run TestBurstRequestsRemainStable -v
SIMPEG_SOAK=1 SIMPEG_SOAK_DURATION=30 SIMPEG_SOAK_STREAMS=16 go test ./internal/httpserver/... -run TestSoakReadPath -v
```

Uji soak bersifat opt-in (`SIMPEG_SOAK=1`) agar pipeline CI tetap cepat. Angka hasil pengukuran dan kekurangannya dicatat di `../docs/PERFORMANCE-BASELINE.md`.

### Perkakas pengukuran terhadap dependensi nyata

`cmd/perfbench` mengukur backend terhadap PostgreSQL, Redis, QStash, dan SMTP sungguhan. Perkakas ini adalah perkakas operator, bukan bagian dari jalur permintaan aplikasi, dan hanya membaca kredensial dari environment.

```bash
export DATABASE_URL="postgres://<user>:<sandi>@<host>:5432/<database>?sslmode=require"

go run ./cmd/perfbench seed     --employees=5000 --months=6 --audit=50000
go run ./cmd/perfbench schema
go run ./cmd/perfbench explain
go run ./cmd/perfbench latency  --iterations=40
go run ./cmd/perfbench e2e      --requests=400 --streams=16
go run ./cmd/perfbench e2e      --soak --duration=60s --streams=16
go run ./cmd/perfbench deps     --redis --qstash --smtp --count=100
go run ./cmd/perfbench slo
```

| Subcommand | Kegunaan |
| --- | --- |
| `seed` | Mengisi database uji dengan data sintetis (pegawai, unit, slip gaji, jejak audit). |
| `schema` | Membandingkan kolom model GORM dengan kolom yang benar-benar ada di database target. |
| `explain` | Menjalankan `EXPLAIN (ANALYZE, BUFFERS)` atas statement yang direkam saat repository produksi berjalan. |
| `latency` | Mengukur distribusi latensi tiap skenario query. |
| `e2e` | Menjalankan binari `cmd/server` asli lalu mengukurnya lewat HTTP. |
| `deps` | Mengukur beban Redis, QStash, dan SMTP. |
| `slo` | Memanggil endpoint pemeriksaan SLO dengan signature QStash seperti callback sungguhan. |

Perintah `deps` menyentuh layanan pihak ketiga. Uji Redis memakai namespace sendiri dan menghapus kuncinya. Uji SMTP mengirim email sungguhan; penerima bawaan adalah alamat akun SMTP sendiri agar uji beban tidak mengirimi pihak ketiga, dan dapat diganti lewat `PERFBENCH_SMTP_RECIPIENTS`. Uji QStash memakai tujuan sekali pakai yang memang tidak dieksekusi.

Jalur volume tinggi dijaga jumlah statement-nya oleh pengujian:

- Impor massal pegawai memeriksa identitas dengan satu query per chunk 250 baris, bukan satu query per baris per kolom (`internal/directory/repository/gorm_import.go`).
- Jejak audit impor ditulis dengan satu statement multi-row lewat `audit.WriteBatch` (`internal/audit/audit.go`).
- Notifikasi penggajian diantrekan dengan satu statement lewat `outbox.EnqueueBatch` (`internal/outbox/gorm_store.go`).

Pengujian penjaga jumlah statement ada di `internal/directory/repository/gorm_import_test.go` dan `internal/payroll/repository/schedule_test.go`. Bentrok impor terhadap data yang sudah ada tetap dilaporkan per kolom (`email`, `nip`, `nidn`, `nidk`) beserta indeks baris yang tepat.
