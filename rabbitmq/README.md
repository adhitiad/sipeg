# RabbitMQ untuk SIMPEG

Folder ini memuat stack broker RabbitMQ yang berdiri sendiri dan mengambil
peran QStash pada deployment self-hosted (docker/podman). Topologi dan jadwal
tidak disimpan di sini — semuanya dideklarasikan dan dijalankan oleh kode
backend agar tidak bisa menyimpang dari konstanta program.

## Isi folder

- `docker-compose.yml` — stack broker: satu layanan `rabbitmq` beserta
  jaringan `simpeg-broker` yang dipakai bersama stack backend.
- `rabbitmq.conf` — pengaturan broker yang dipasang ke container sebagai
  `/etc/rabbitmq/conf.d/10-simpeg.conf`.
- File ini — penjelasan peran broker dan cara mengoperasikannya.

## Pemisahan dari backend

Broker tidak lagi berada dalam satu `docker-compose.yml` dengan backend.
Alasannya: pengembang dapat memulai dan mematikan broker tanpa menyentuh
stack backend, dan backend dapat dijalankan native (`go run`) sambil
tersambung ke broker container.

- Menjalankan broker saja (pengembangan):

  ```bash
  cd rabbitmq
  docker compose up -d
  ```

  Lalu jalankan backend native dari `backend/` dengan
  `RABBITMQ_URL=amqp://simpeg:simpeg@localhost:5672/` (lihat `.env.example`).

- Deployment satu server: broker dan backend tetap berjalan pada satu host
  tetapi sebagai dua stack compose. Stack broker membuat jaringan
  `simpeg-broker`; stack backend (`docker-compose.yml` pada root) bergabung
  ke jaringan itu sebagai external network, sehingga layanan `backend` dan
  `jobsched` menyambung ke host `rabbitmq` pada jaringan bersama.

  ```bash
  docker compose -f rabbitmq/docker-compose.yml up -d   # broker lebih dulu
  docker compose up -d --build                          # migrasi + penjadwal + backend
  ```

  Saat mematikan, hentikan stack backend terlebih dahulu; jaringan dan
  volume milik stack broker.

## Topologi yang dibuat backend

Semua dideklarasikan idempoten saat koneksi dibuka (paket `internal/mq`):

| Nama | Jenis | Peran |
|------|-------|-------|
| `simpeg.jobs` | direct exchange | menerima pesan pekerjaan; routing key = kind pekerjaan |
| `simpeg.jobs` | durable queue | antrean eksekusi; dead-letter ke `simpeg.jobs.dlx` |
| `simpeg.jobs.dlx` | direct exchange | perantara pesan gagal |
| `simpeg.jobs.dead` | durable queue | tempat pesan gagal untuk diperiksa operator |

Kind pekerjaan yang dikenal: `audit.cleanup` (harian 02:00 WIB),
`payroll.notify` (harian 02:05 WIB), dan `slo.evaluate` (tiap 5 menit,
hanya bila `SLO_EVALUATE_ENABLED=true` pada penjadwal).

## Siapa memproduksi dan mengonsumsi

- `cmd/jobsched` adalah penjadwal: tick cron dalam UTC menerbitkan satu
  pesan per tick dengan publisher confirm.
- Server (`cmd/server`) mengonsumsi antrean, menjalankan pekerjaan yang sama
  dengan callback QStash, lalu mengakui pesan hanya setelah sukses. Pesan
  gagal pindah ke `simpeg.jobs.dead`.
- Pekerjaan idempoten, jadi redelivery tidak menghasilkan efek ganda.

## Verifikasi manual

Kirim satu pesan uji tanpa menunggu tick cron (dari root repo):

```bash
docker compose run --rm jobsched publish audit.cleanup
docker compose logs backend | grep mq_job
docker compose -f rabbitmq/docker-compose.yml exec rabbitmq rabbitmqctl list_queues name messages
```

UI manajemen tersedia pada `http://localhost:15672` dari host (porta
dipetkan hanya untuk debugging; jangan terekspos di production).

## Keamanan

- Kredensial bawaan compose adalah `simpeg`/`simpeg` — ganti melalui
  `RABBITMQ_USER`/`RABBITMQ_PASSWORD` sebelum dipakai di luar komputer
  lokal.
- Porta host 5672 ada semata untuk pengembangan native dan debugging;
  hapus pemetaan porta tersebut pada deployment produksi yang ketat dan
  biarkan lalu lintas hanya di jaringan `simpeg-broker`.
- Tidak ada pesan yang memuat data pribadi: payload hanya `{"kind": ...}`.
