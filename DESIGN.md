# DESIGN.md — SIMPEG frontend (draft)

> Status: **DRAFT — menunggu persetujuan product owner.** Dokumen ini mencatat
> arah desain yang dipakai pada implementasi saat ini di `frontend/`. Token
> dan aturan tidak boleh dipakai sebagai rujukan final sebelum product owner
> menyetujui revisinya.

Sumber token langsung: `frontend/src/styles.css`, `frontend/src/styles/components.css`.
Dokumen ini merangkumnya; bila keduanya bertentangan, stylesheet yang menang
dan dokumen ini harus diperbaiki.

## Karakter produk

SIMPEG adalah sistem kepegawaian untuk lingkungan instansi/perguruan tinggi.
Desainnya harus terasa tenang, resmi, dan dapat diandalkan — bukan seperti
produk konsumen. Kalimat keputusan:

- **Tenang, bukan sepi.** Latar hijau pucat dengan dua aksen radial (lagoon
  dan palem) memberi kedalaman tanpa keramaian. Tidak ada gradien teks,
  kaca buram menyeluruh, atau neon.
- **Daerah besar rata, satu aksen per tampilan.** Lagoon (`--lagoon` /
  `--lagoon-deep`) hanya untuk aksen kecil: titik logo, underline nav,
  fokus ring. Palem (`--palm` / `--kicker`) untuk kicker dan aksi utama.
- **Konten resmi.** Judul tampilan `Fraunces` (serif, opsz 9..144, 500/700);
  isi `Manrope` (400–800). Perpaduan serif + sans dipakai untuk membedakan
  suara editorial (judul) dari suara operasional (data dan label).

## Palet (light bawaan, dark alternatif)

Bawaan adalah **light**. `data-theme="light|dark"` ditulis inline oleh skrip
di `__root.tsx` sebelum hidrasi; `.dark` mengaktifkan palet gelap.

### Light

| Token                            | Nilai                           | Pemakaian                                    |
| -------------------------------- | ------------------------------- | -------------------------------------------- |
| `--sea-ink`                      | `#173a40`                       | Teks utama, foreground                       |
| `--sea-ink-soft`                 | `#416166`                       | Teks sekunder, muted-foreground              |
| `--lagoon`                       | `#4fb8b2`                       | Aksen: titik logo, nav underline, ring fokus |
| `--lagoon-deep`                  | `#246f76`                       | Teks tautan, avatar fallback                 |
| `--palm`                         | `#2f6a4a`                       | Aksi utama, kicker                           |
| `--kicker`                       | `#2f6a4a`                       | Label kicker (uppercase, letterspaced)       |
| `--foam`                         | `#f3faf5`                       | Gradien atas body                            |
| `--bg-base`                      | `#e7f3ec`                       | Latar dasar body                             |
| `--surface` / `--surface-strong` | `rgba(255,255,255,.76)` / `.92` | Shell/island, surface kartu                  |
| `--line`                         | `rgba(23,58,64,.16)`            | Border halus                                 |
| `--chip-bg` / `--chip-line`      | putih 82% / palem 22%           | Chip, tombol ikon, header pill               |

### Dark

| Token                            | Nilai                        | Pemakaian     |
| -------------------------------- | ---------------------------- | ------------- |
| `--sea-ink`                      | `#d7ece8`                    | Teks utama    |
| `--sea-ink-soft`                 | `#afcdc8`                    | Teks sekunder |
| `--lagoon`                       | `#60d7cf`                    | Aksen terang  |
| `--lagoon-deep`                  | `#8de5db`                    | Tautan        |
| `--palm`                         | `#6ec89a`                    | Aksi utama    |
| `--kicker`                       | `#b8efe5`                    | Kicker        |
| `--foam`                         | `#101d22`                    | Gradien atas  |
| `--bg-base`                      | `#0a1418`                    | Latar dasar   |
| `--surface` / `--surface-strong` | `rgba(16,30,34,.84)` / `.94` | Shell         |
| `--line`                         | `rgba(141,229,219,.2)`       | Border halus  |

Kontras teks normal ditargetkan ≥ 4.5:1 (WCAG AA) pada kedua mode; kicker
hanya dipakai untuk label pendek di atas latar yang memenuhinya.

## Tipografi

- **Fraunces** hanya untuk judul tampilan (`.display-title`): `opsz` variabel,
  weight 500/700, `letter-spacing` normal, ukuran clamp minimum 2rem.
- **Manrope** untuk seluruh isi: 400 (teks panjang), 500 (label), 600–800
  (nav, tombol, nama).
- Skala: hero `clamp(2rem…3rem)`, h2 `1.5rem`, body `0.95–1rem`, caption
  `0.69–0.75rem` (kicker 0.69rem, uppercase, `0.16em`).
- `code` memakai border `--line` dan `--surface-strong`; `pre code` datar.

## Tata letak

- `.page-wrap`: lebar `min(1080px, calc(100% - 2rem))`, selalu ter-center.
  Halaman tidak pernah memakai lebar penuh tanpa gutter.
- `.island-shell`: kartu shell ber-border dengan gradient 165° dari
  `--surface-strong` ke `--surface`, blur 4px, dan bayangan tipis.
  Dipakai untuk panel konten (login, error, ruang kerja). **Tidak** dipakai
  sebagai motif berulang di seluruh halaman.
- `.feature-card`: varian island untuk kartu informasi; hover hanya mengubah
  `border-color` (palm 35% campuran), tanpa terangkat/terangkat-shadow.
- Hero landing: dua kolom pada ≥ `md` (teks + panel data sintetis), satu
  kolom di ponsel. Tidak ada angka/statistik fiktif yang terlihat seperti
  data produksi; seluruh isi contoh ditandai sebagai contoh.
- Mobile-first: 375px adalah lebar acuan; tidak boleh ada overflow horizontal
  (diuji di Playwright).

## Komponen utama

- **Header**: logo titik lagun + "SIMPEG", nav, saklar bahasa `id | en`
  (`aria-pressed`, `role="group"`), toggle tema (ikon Moon/Sun,
  `aria-label` sesuai bahasa aktif), lalu area sesi: tombol "Masuk" bila
  belum login; avatar inisial (fallback `--lagoon-deep`) dengan menu:
  "Ruang kerja" + "Keluar". Menu menutup dengan Escape dan klik luar.
- **Footer**: tiga kolom ringkas (tentang, legal, kontak) dengan tautan ke
  `/about`, `/privacy`, `/terms`, `/terms-of-reference`.
- **Toaster**: sonner, `bottom-right`, mengikuti tema aktif; dipakai untuk
  hasil masuk/keluar dan galat API yang dapat dipulihkan.
- **Login**: email + kata sandi, zod + pesan berbahasa aktif, tanpa tombol
  "daftar" (akun dibuat administrator).
- **Indikator memuat** (`LoadingBars`, `LoadingRadar`): dua bentuk yang
  berbeda untuk dua jenis pekerjaan, keduanya memakai token produk dan tidak
  memakai warna literal Tailwind.
  - `LoadingBars` = petak dengan tepi gradien lagoon-ke-palem berputar dan
    batang di dalam yang memantul bergantian. Dipakai untuk keadaan memuat
    umum: halaman, sesi, metrik SLO, dan tombol masuk.
  - `LoadingRadar` = radar dengan lengan berputar menyapu cincin konsentris.
    Dipakai khusus pada daftar pegawai, karena yang sedang berjalan di sana
    adalah penyaringan unit, bukan sekadar perpindahan halaman.
  - Tanpa glow, tanpa blur, tanpa drop shadow. Gradien conic pada `LoadingBars`
    dibatasi pada tepi petak sehingga tidak menjadi "gradien utama" layar.
  - Gerak hanya berjalan saat data memang dimuat dan dimatikan oleh
    `prefers-reduced-motion: reduce`; bentuk radar dan petak tetap terbaca
    sebagai tanda proses berjalan.
  - Aksesibilitas: `role="status"` + `aria-busy` + `aria-label` pada satu
    elemen. Pemanggil yang sudah menampilkan teks status memakai prop
    `decorative`, lalu indikator disembunyikan dari pembaca layar agar teks
    tidak dibacakan dua kali.

## Tiga dial desain

| Dial   | Nilai          | Arti                                                                                                                                                                                          |
| ------ | -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ENERGY | Rendah (2/10)  | Tidak ada ledakan warna/ilustrasi; kedalaman lewat dua radial latar.                                                                                                                          |
| RHYTHM | Sedang (4/10)  | Ritme section 3–5rem; kicker → judul → isi sebagai pola berulang.                                                                                                                             |
| MOTION | Minimal (2/10) | Hanya `rise-in` (600ms `cubic-bezier(.16,1,.3,1)`) saat mount dan underline nav (170ms); satu-satunya pengecualian adalah indikator memuat yang memang berjalan selama proses masih berjalan. |

`prefers-reduced-motion: reduce` mematikan keduanya. Tidak ada animasi
scroll, parallax, atau auto-play.

## Alasan satu baris atas keputusan visual

| Keputusan                                                                      | Alasan                                                                                                             |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------ |
| Radial `--hero-a` di sudut kiri-atas body                                      | satu sumber kedalaman; tanpa foto/ilustrasi, latar tetap tenang di belakang konten resmi                           |
| `island-shell`: gradient 165° dari `--surface-strong` ke `--surface`, blur 4px | panel naik satu tingkat di atas latar datar; lebih dari 4px berubah jadi kaca buram, arah yang ditolak produk ini  |
| Radius `0.75rem` (`--radius`)                                                  | sudut cukup lembut untuk ramah, cukup kecil untuk tetap resmi; semua turunan sm/md/lg/xl mengikuti nilai ini       |
| `rise-in` 600ms `cubic-bezier(.16,1,.3,1)`                                     | satu gerakan masuk saat mount; lengkung panjang tanpa pantulan; dimatikan `prefers-reduced-motion`                 |
| Fokus `outline: 2px solid var(--ring)` + offset 2px                            | cincin terlihat di kedua mode dan tidak menutup konten                                                             |
| Loader batang untuk keadaan memuat umum, radar untuk daftar pegawai            | dua jenis pekerjaan, dua tanda yang berbeda; radar menyiratkan penyaringan unit, bukan sekadar perpindahan halaman |
| Warna loader dari `--lagoon`, `--palm`, `--sea-ink`                            | tidak menambah aksen baru di luar palet produk; tetap terbaca pada light dan dark                                  |

## Bahasa dan tema

- Default locale `id`; `en` selalu tersedia; teks hanya dari kamus
  (`src/i18n/messages/`). Bila menambah teks, tambahkan ke kedua katalog.
- Bawaan tema light; toggle memakai `setThemeMode` yang menulis localStorage
  `theme` + atribut `data-theme` pada `<html>`.
- Ikon saja tanpa label tidak boleh; setiap tombol ikon memiliki `aria-label`
  berbahasa aktif.

## Yang tidak boleh terjadi (anti-slop / anti pola AI)

- Tidak ada gradient teks, border gradien, atau "glassmorphism" menyeluruh.
- Tidak ada emoji di UI resmi, tidak ada bullet dengan ikon acak.
- Tidak ada data pegawai fiktif yang terlihat nyata di UI; contoh selalu
  sintetis dan bertanda.
- Tidak ada klaim status ("production-ready", "99.9% uptime") tanpa bukti.
- Tidak ada teks filler ("Lorem ipsum", kata-kata pemasaran generik) di
  halaman hukum/legal.
- Tidak menambahkan warna aksen baru tanpa alasan produk; lagoon/palm/sea-ink
  sudah memadai.

## Aksesibilitas

- Fokus terlihat: `:focus-visible` dengan `outline: 2px solid var(--ring)`.
- Target sentuh ≥ 44px pada ponsel (tombol ikon `h-11 w-11` pada < `sm`).
- Hierarki heading tunggal per halaman; `main`/`banner`/`contentinfo` selalu
  ada lewat layout root.
- Kontras teks AA pada kedua mode; status (error/sukses) tidak disampaikan
  hanya lewat warna — selalu ada teks.

## Checklist review desain

- [ ] Halaman tetap terbaca pada 375px tanpa overflow horizontal
- [ ] Light dan dark sama-sama layak (bukan salah satunya "rendisi kedua")
- [ ] Tidak ada string keras; semua lewat kamus `id`/`en`
- [ ] Satu aksen utama per layar; palet sesuai token
- [ ] Keyboard: Tab/Shift+Tab/Escape mencakup semua aksi
- [ ] `prefers-reduced-motion` menghormati animasi
