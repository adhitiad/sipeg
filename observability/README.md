# Observability stack (self-hosted, $0) — SIMPEG

Stack observability untuk target: **Docker** (backend service `backend:8080`), **RabbitMQ self-hosted** (stack `rabbitmq/`, jaringan `simpeg-broker`), log aplikasi **JSON ke stdout**, dan **Supabase Free**. Pemilik: **adhitiad (admin)**. Rujukan: `docs/DECISIONS.md` `DEC-065`, `docs/RUNBOOK.md` §9/§9A.

> **Biaya: $0.** Seluruh komponen open-source dan berjalan lokal/self-hosted. Jangan menambahkan layanan berbayar (Grafana Cloud berbayar, SaaS APM berbayar, PITR Supabase berbayar, dsb.).  
> **Status:** templat siap-disesuaikan; wiring, penerima alert nyata, dan verifikasi end-to-end di target **tetap wajib dibuktikan**.

## Komponen & berkas

| Komponen          | Peran                                | Berkas                                 |
| ----------------- | ------------------------------------ | -------------------------------------- |
| Prometheus        | Pengumpulan metrik + evaluasi aturan | `prometheus.yml`, `alerts.yml`         |
| Alertmanager      | Routing/notifikasi alert             | `alertmanager.yml`                     |
| Grafana           | Dasbor metrik & log                  | `docker-compose.yml`                   |
| Loki              | Agregasi log (retensi 30 hari)       | `loki-config.yml`                      |
| Promtail          | Koleksi log Docker stdout JSON       | `promtail-config.yml`                  |
| node-exporter     | Saturasi host                        | `docker-compose.yml`                   |
| cAdvisor          | Saturasi container                   | `docker-compose.yml`                   |
| blackbox-exporter | Probe `/health`, `/ready`            | `docker-compose.yml`, `prometheus.yml` |

## Target & sumber sinyal (disesuaikan ke stack)

- **Backend:** probe `http://backend:8080/health` dan `http://backend:8080/ready` via jaringan `simpeg-broker` (stack utama `docker-compose.yml`).
- **RabbitMQ:** scrape `rabbitmq:15692` (plugin `rabbitmq_prometheus`; verifikasi plugin aktif pada broker/port terjangkau).
- **Log:** Promtail membaca Docker stdout JSON; field `request_id`, `level`, `msg`, `path`, `status` diparse (lihat `promtail-config.yml`). Retensi agregasi awal **30 hari** (`loki-config.yml`).
- **Sinyal admin aplikasi (butuh auth):** `GET /api/v1/admin/slo/metrics` (`slo:read`, butuh `SLO_ENABLED=true` untuk data) dan `GET /api/v1/admin/ops/status` (`ops:read`) — agregat outbox/backlog & kegagalan notifikasi. Untuk otomasi $0 gunakan proxy/exporter internal yang menyuntikkan token dari secret host, atau polling terjadwal aman; **jangan** menaruh token di repo.
- **Exporter aman (tanpa token):** service `metricsexport` (port 9101) membaca agregat outbox/SLO/SMTP/dependency eksternal langsung dari PostgreSQL dan mengeksposnya sebagai metrik Prometheus. Scrape job `simpeg-exporter`. Kredensial dibaca dari env (DATABASE_URL, UPSTASH_*, SMTP_*), tidak ada di repo. Verifikasi dependensi terautentikasi memakai `go run ./cmd/depcheck`.
- **Supabase (Free):** gunakan metrik yang tersedia di dashboard/project; PITR umumnya tidak tersedia — verifikasi kemampuan tier/project.

## Cara pakai ($0)

1. **Prasyarat:** stack `rabbitmq/` dan stack utama sudah berjalan (jaringan `simpeg-broker` ada). Docker & Docker Compose terpasang.
2. **Konfigurasi:** salin `observability/.env.example` ke `observability/.env` (lokal, jangan commit) dan ganti kata sandi Grafana. Isi penerima nyata di `alertmanager.yml` (webhook internal/email SMTP yang tersedia) lalu rencanakan uji tembak.
3. **Sesuaikan (jika perlu):** nama service/port backend atau RabbitMQ jika berbeda, port host (`PROMETHEUS_PORT`/`GRAFANA_PORT`/`LOKI_PORT`), dan aturan alert (`alerts.yml`) sesuai baseline.
4. **Jalankan:** `cd observability && docker compose up -d`
5. **Verifikasi:** Prometheus targets `UP` (termasuk blackbox/rabbitmq bila aktif), Grafana dapat query Prometheus & Loki, log container tampil, probe backend sehat, dan alert uji tembak diterima penerima.

## Checklist penutupan (harus dibuktikan di target)

- [ ] Pengumpulan metrik/log & dasbor end-to-end aktif (saturasi host/container, health backend/RabbitMQ, sinyal aplikasi yang relevan).
- [ ] Aturan alert aktif di Alertmanager/Prometheus dan **uji tembak** terkirim ke penerima/eskalasi yang ditetapkan.
- [ ] Retensi/akses log agregasi (30 hari awal) aktif dan penelusuran `request_id` terverifikasi.
- [ ] Backup/restore (dan batas PITR sesuai tier Supabase Free) diuji pada target terpisah/representatif; hasil dicatat (`docs/backup-restore-drill.md`).

> UI/metrik jangan diekspos ke publik tanpa proteksi. Kredensial/token hanya di env/secret host, tidak di repo.
