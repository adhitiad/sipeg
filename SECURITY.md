# Kebijakan Keamanan SIMPEG

## Melaporkan kerentanan

Jangan membuka issue publik untuk kerentanan keamanan. Kirim laporan ke
`adhitiadwima@gmail.com` (admin sistem) dengan subjek `[SECURITY] SIMPEG`,
sertakan langkah reproduksi, dampak, dan (bila ada) X-Request-ID.

## Praktik keamanan

- Rahasia (DATABASE_URL, PII_ENCRYPTION_KEY, SMTP_PASS, dll.) hanya di
  environment/secret manager, tidak di repo.
- Backend menerapkan deny-by-default, RLS default-deny, enkripsi PII level
  aplikasi (AES-256-GCM + blind index HMAC-SHA256).
- Jejak audit append-only; penghapusan hanya via retensi terjadwal.
- Notifikasi gaji tidak pernah memuat nominal dan hanya dikirim setelah
  pembayaran terkonfirmasi.