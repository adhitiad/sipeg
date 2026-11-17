# Observability stack (self-hosted, $0) — SIMPEG

Stack observability untuk target Docker + RabbitMQ self-hosted (`simpeg-broker`), stdout JSON, dan Supabase Free. Pemilik: **adhitiad (admin)**. Rujukan: `docs/DECISIONS.md` DEC-065, `docs/RUNBOOK.md` §8E/§9/§9A.

> **Biaya software: $0.** Semua komponen open-source dan dijalankan self-hosted. Biaya host, disk, backup off-site, domain, dan konektivitas tetap mengikuti infrastruktur.
>
> **Status:** template dan validasi konfigurasi tersedia. Instalasi pada host target, penerima alert nyata, dan backup/restore project Supabase aktual **belum dibuktikan oleh repo/CI**.

## Komponen

| Komponen | Fungsi | Konfigurasi |
|---|---|---|
| Prometheus 3.16 | Scrape metrics dan evaluasi alert | `prometheus.yml`, `alerts.yml` |
| Alertmanager 0.35 | Routing notifikasi | `alertmanager.yml` |
| Grafana 13.3 | Dashboard yang diprovisikan | `grafana/` |
| Loki 3.6 | Penyimpanan log dengan retensi 30 hari | `loki-config.yml` |
| Grafana Alloy 1.21 | Koleksi Docker JSON logs | `alloy-config.alloy` |
| node-exporter | CPU, RAM, filesystem, jaringan host | Compose |
| cAdvisor | CPU/RAM container | Compose |
| blackbox-exporter | HTTP probe backend dan TCP probe dependency | `blackbox.yml` |
| RabbitMQ Prometheus plugin | Metrics broker dan queue tertentu | `rabbitmq/enabled_plugins` |

Promtail **tidak digunakan**: Promtail telah mencapai EOL pada 2 Maret 2026 dan digantikan Grafana Alloy. Alloy membaca Docker JSON log files dari `/var/lib/docker/containers` read-only; ia tidak memerlukan Docker socket. Sesuaikan mount path bila Docker menggunakan `data-root` non-default. Alloy hanya memberi label `level`; `request_id` tetap berada di isi log agar kardinalitas Loki tidak meledak.

## Sinyal yang dikumpulkan

- **Host/container:** CPU host, available memory, filesystem free, dan resource container melalui node-exporter/cAdvisor.
- **Backend:** HTTP probe `/health` (liveness) dan `/ready` (readiness). Readiness backend saat ini memeriksa PostgreSQL wajib.
- **RabbitMQ:** aggregate node/runtime metrics melalui `/metrics`; filtered per-queue metrics melalui `/metrics/detailed` untuk hanya `simpeg.jobs` dan `simpeg.jobs.dead`. Family `queue_coarse_metrics` menyediakan ready, unacked, dan total depth; family `queue_metrics` menyediakan consumer count serta head-message timestamp. Family `queue_metrics` lebih mahal dihitung, tetapi regex queue membatasi cakupannya hanya pada dua antrean SIMPEG. Alert backlog, zero consumer, DLQ non-empty, detailed scrape down, dan umur pesan tertua aktif. Umur pesan dihitung dari timestamp Unix pesan yang dipublikasikan backend; nilainya dapat tidak tersedia jika pesan tidak memiliki timestamp atau tidak paged-in.
- **External dependency (opsional):** SMTP submission, Supabase PostgreSQL, dan Upstash Redis TLS dapat diuji konektivitas TCP setelah host:port aktual ditambahkan ke `targets/external-tcp.yml`. TCP probe hanya membuktikan koneksi port, bukan autentikasi SMTP/Redis atau query database.
- **Log aplikasi:** Docker stdout JSON → Alloy → Loki. Compose root, RabbitMQ, dan observability menetapkan driver json-file dengan rotasi maksimum 20 MB × 5 file per container. Ini spool lokal, bukan retensi 30 hari; Alloy perlu tetap berjalan agar log tersalin ke Loki. Gunakan LogQL `{job="docker"} | json | request_id="<id>"` untuk korelasi request; gunakan `level` sebagai label.
- **Sinyal aplikasi yang belum otomatis discrape:** `GET /api/v1/admin/ops/status` dan `GET /api/v1/admin/slo/metrics` tetap dilindungi RBAC. Outbox pending/failed/oldest age serta agregat SLO **belum diekspor sebagai Prometheus metrics**. Jangan memasukkan token admin statis ke file konfigurasi Prometheus; tahap berikutnya adalah exporter internal dengan identitas khusus dan secret file.

RabbitMQ plugin `rabbitmq_prometheus` diaktifkan eksplisit dari `rabbitmq/enabled_plugins`. Jangan mengandalkan plugin bawaan image. Queue-level scrape sengaja difilter agar tidak mengambil metrik per-object seluruh broker.

## Pemasangan

1. Salin root `.env.example` menjadi `.env`; isi RABBITMQ_USER dan RABBITMQ_PASSWORD dengan nilai unik. Buat password hex URL-safe, misalnya `openssl rand -hex 32`. Jangan memakai fallback simpeg:simpeg.
2. Salin `observability/.env.example` menjadi `observability/.env`. Tetapkan `GRAFANA_ADMIN_PASSWORD` yang kuat dan unik. Jangan commit kedua file `.env`.
3. Ganti webhook test sink di `alertmanager.yml` dengan penerima produksi yang disetujui. Alamat `host.docker.internal:9999` adalah endpoint sink uji lokal, **bukan penerima produksi**.
4. Bila perlu pantau SMTP/Supabase/Upstash, tambahkan host:port aktual ke `targets/external-tcp.yml`. Jangan menulis password, API key, atau token di file target.
5. Jalankan broker dahulu agar jaringan `simpeg-broker` tersedia, lalu backend dan observability:
   ```bash
   docker compose -f rabbitmq/docker-compose.yml up -d
   docker compose up -d --build
   cd observability
   docker compose --env-file .env up -d
   ```
6. Validasi file sebelum start:
   ```powershell
   .\validate.ps1
   ```
7. Verifikasi Prometheus Targets: `blackbox`, `rabbitmq`, `rabbitmq-queues`, `node-exporter`, `cadvisor`, dan `prometheus` harus `UP`; job `blackbox-external` baru memiliki target setelah target TCP diisi. Grafana otomatis mendapat datasource Prometheus/Loki dan dashboard **SIMPEG — Infrastructure & Queue**.
8. Lakukan uji tembak alert ke penerima aktual, kemudian catat waktu firing/resolved, identitas penerima, dan eskalasi pada runbook. Jangan menandai alert produksi aktif berdasarkan sink lokal.

## Alert awal

- Backend `/health` atau `/ready` gagal 2 menit: critical.
- Host CPU >85% selama 5 menit, host memory available <10% selama 5 menit, container memory >85%, filesystem free <15% selama 5 menit: warning.
- RabbitMQ jobs ready >100 selama 5 menit: warning; umur pesan tertua >300 detik selama 5 menit: warning.
- Dead-letter `simpeg.jobs.dead` berisi pesan selama 1 menit: critical.
- Queue jobs memiliki backlog tetapi consumer count 0 selama 2 menit: critical.
- TCP dependency eksternal gagal 2 menit: warning.

Threshold wajib disesuaikan memakai baseline produksi. Aturan outbox/SLO belum aktif karena aggregator aplikasi masih berupa endpoint admin yang terproteksi.

## Retensi dan keamanan log

- Loki retensi awal **720 jam / 30 hari**, diaktifkan oleh compactor. Prometheus retensi **15 hari**; retensi metrics berbeda dari retensi log.
- Docker json-file rotation: **20 MB × 5 file per container** (maksimum sekitar 100 MB buffer lokal per container). Ini bukan jaminan 30 hari; kehilangan log lokal masih mungkin bila Alloy berhenti cukup lama atau log rate tinggi.
- Log agregasi terpisah dari audit DB `AUDIT_RETENTION_DAYS` dan `SLO_RETENTION_DAYS`.
- Grafana, Prometheus, Alertmanager, Loki, dan blackbox host ports di-bind ke `127.0.0.1`. Jangan expose UI/metrics ke internet tanpa reverse proxy + autentikasi.
- Alloy hanya memiliki akses read-only ke Docker log files. cAdvisor memerlukan `privileged` dan akses Docker host untuk mengukur container; evaluasi kembali bila host memakai security policy yang lebih ketat.
- Jangan jadikan `request_id`, pesan bebas, alamat email, atau ID pegawai sebagai label Loki/Prometheus berkardinalitas tinggi.

## Backup Supabase Free

Supabase Free **tidak menyediakan automatic daily database backups maupun PITR**. Lakukan ekspor terjadwal menggunakan Supabase CLI `db dump`/pg_dump dan simpan salinan off-site yang terenkripsi. Restore drill harus menuju project/database disposable terpisah; file dalam Supabase Storage API dan berkas Cloudinary memerlukan strategi backup objek terpisah. Rinciannya ada di `docs/RUNBOOK.md` §8E dan `docs/backup-restore-drill.md`.

## Checklist bukti pada infrastruktur target

- [ ] Semua scrape targets yang dipakai deployment `UP`; dashboard menunjukkan data nyata.
- [ ] Queue backlog, consumer count, dan dead-letter RabbitMQ terlihat; simulasi pesan gagal menghasilkan alert.
- [ ] Dependency SMTP/Supabase/Redis yang relevan diisi ke file target dan diuji sesuai protokol aplikasinya.
- [ ] Outbox/SLO metrics diekspor melalui exporter internal aman (belum tersedia).
- [ ] Alertmanager memakai penerima produksi yang benar dan uji tembak + eskalasi direkam.
- [ ] Penelusuran `request_id` dan retensi 30 hari diverifikasi setelah log melewati masa retensi.
- [ ] Backup Free terjadwal, off-site, dan restore ke project disposable diuji; RPO/RTO aktual dicatat.

> Templat ini bukan bukti deployment. Status checklist produksi hanya dapat diubah setelah bukti dari host dan project Supabase target dilampirkan.
