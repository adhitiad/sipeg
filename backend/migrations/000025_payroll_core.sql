-- USULAN SKEMA — 2026-11-14 (DEC-066). JANGAN DITERAPKAN pada database mana pun
-- sebelum pemilik produk meratifikasi aturan bisnis (DEC-065) dan menyetujui
-- penerapannya. Migrasi ini belum diverifikasi terhadap database target; nama
-- tabel/kolom, tipe, dan index masih dapat berubah. GORM model (registry)
-- sengaja belum didaftarkan dan endpoint belum tersedia.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "KategoriStatusPegawai" AS ENUM (
        'DOSEN_TETAP_YAYASAN', 'DOSEN_DPK', 'DOSEN_LB', 'TENDIK_TETAP', 'TENDIK_KONTRAK'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisBPJS" AS ENUM ('KESEHATAN', 'KETENAGAKERJAAN');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "PtkpKategoriTER" AS ENUM ('A', 'B', 'C');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisKontrak" AS ENUM ('PKWT', 'PKWTT');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisSk" AS ENUM (
        'PENGANGKATAN', 'MUTASI', 'PHK', 'JABATAN_STRUKTURAL', 'PENUGASAN'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "TingkatSk" AS ENUM ('YAYASAN', 'REKTOR');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- ═══════════ 1. Kategori status pegawai ═══════════

-- Taxonomy kepegawaian yayasan. `kategoriPegawai` yang sudah ada (golongan/usia
-- pensiun) dipertahankan; kategori ini menentukan aturan gaji, kontrak, dan
-- pelaporan. `gajiDariNegara` = true untuk dosen DPK (gaji dari negara,
-- dikelola via LLDikti); kategori tersebut tidak menerima gaji pokok yayasan.
CREATE TABLE IF NOT EXISTS "kategoriStatusPegawai" (
    "id"               uuid PRIMARY KEY,
    "kode"             text NOT NULL UNIQUE,
    "nama"             text NOT NULL,
    "deskripsi"        text,
    "gajiDariNegara"   boolean NOT NULL DEFAULT false,
    "butuhKontrak"     boolean NOT NULL DEFAULT false,
    "wajibBpjs"        boolean NOT NULL DEFAULT true,
    "bupSesuaiYayasan" boolean NOT NULL DEFAULT true,
    "aktif"            boolean NOT NULL DEFAULT true,
    "createdAt"        timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"        timestamp(3) NOT NULL DEFAULT now()
);

INSERT INTO "kategoriStatusPegawai" ("id", "kode", "nama", "deskripsi", "gajiDariNegara", "butuhKontrak", "wajibBpjs", "bupSesuaiYayasan") VALUES
  ('d0000000-0000-0000-0000-000000000001', 'DOSEN_TETAP_YAYASAN', 'Dosen tetap yayasan', 'Dosen tetap yayasan; gaji pokok dari yayasan; jenjang fungsional.', false, false, true, true),
  ('d0000000-0000-0000-0000-000000000002', 'DOSEN_DPK',           'Dosen DPK',              'Dosen dipekerjakan ke instansi; gaji dari negara, pelaporan via LLDikti; yayasan tidak membayar gaji pokok.', true,  false, true, true),
  ('d0000000-0000-0000-0000-000000000003', 'DOSEN_LB',            'Dosen tidak tetap / LB', 'Dosen luar biasa/tidak tetap; honor per SKS; kontrak per periode.', false, true,  true, false),
  ('d0000000-0000-0000-0000-000000000004', 'TENDIK_TETAP',        'Tenaga kependidikan tetap', 'Tendik tetap yayasan; gaji pokok dan jenjang fungsional.', false, false, true, true),
  ('d0000000-0000-0000-0000-000000000005', 'TENDIK_KONTRAK',      'Tenaga kependidikan kontrak', 'Tendik kontrak; PKWT; aturan pesangon sesuai PP 35/2021.', false, true,  true, false)
ON CONFLICT ("kode") DO NOTHING;

-- Pegawai dikaitkan ke satu kategori status. Nullable sampai migrasi data
-- disetujui; pengisian dilakukan bertahap.
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS "kategoriStatusId" uuid REFERENCES "kategoriStatusPegawai" ("id");

-- ═══════════ 2. Skala gaji pokok yayasan ═══════════

-- Skala gaji pokok yayasan berlaku per periode (effective-dated). Satu skala
-- berisi tingkat nominal per kategori status.
CREATE TABLE IF NOT EXISTS "skalaGaji" (
    "id"            uuid PRIMARY KEY,
    "kode"          text NOT NULL UNIQUE,
    "nama"          text NOT NULL,
    "berlakuMulai"  date NOT NULL,
    "berlakuSelesai" date,
    "aktif"         boolean NOT NULL DEFAULT true,
    "createdAt"     timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"     timestamp(3) NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS "skalaGajiDetail" (
    "id"               uuid PRIMARY KEY,
    "skalaGajiId"      uuid NOT NULL REFERENCES "skalaGaji" ("id") ON DELETE CASCADE,
    "kategoriStatusId" uuid NOT NULL REFERENCES "kategoriStatusPegawai" ("id"),
    "tingkat"          integer NOT NULL CHECK ("tingkat" >= 0),
    "nominal"          double precision NOT NULL CHECK ("nominal" >= 0),
    "createdAt"        timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"        timestamp(3) NOT NULL DEFAULT now(),
    UNIQUE ("skalaGajiId", "kategoriStatusId", "tingkat")
);
CREATE INDEX IF NOT EXISTS skala_gaji_detail_kategori_idx ON "skalaGajiDetail" ("kategoriStatusId");

-- ═══════════ 3. SK (surat keputusan) — yayasan vs rektor ═══════════

-- Yayasan menerbitkan SK pengangkatan/mutasi/PHK; rektor menerbitkan SK
-- jabatan struktural dan penugasan. `TingkatSk` menentukan penerbit.
CREATE TABLE IF NOT EXISTS "suratKeputusan" (
    "id"            uuid PRIMARY KEY,
    "nomorSk"       text NOT NULL UNIQUE,
    "jenis"         "JenisSk" NOT NULL,
    "penerbit"      "TingkatSk" NOT NULL,
    "userId"        uuid REFERENCES "user" ("id") ON DELETE SET NULL,
    "homebaseId"    uuid REFERENCES "Homebase" ("id") ON DELETE SET NULL,
    "tanggalSk"     date NOT NULL,
    "mulaiBerlaku"  date,
    "selesaiBerlaku" date,
    "fileUrl"       text,
    "metadata"      jsonb NOT NULL DEFAULT '{}'::jsonb,
    "createdAt"     timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"     timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS surat_keputusan_user_idx ON "suratKeputusan" ("userId");
CREATE INDEX IF NOT EXISTS surat_keputusan_jenis_idx ON "suratKeputusan" ("jenis", "penerbit");

-- ═══════════ 4. Honor mengajar per SKS (dosen LB) ═══════════

-- Tarif honor per SKS per periode berlaku. Kebijakan tarif baru berlaku
-- mulai `berlakuMulai` dan tidak menulis ulang periode berjalan.
CREATE TABLE IF NOT EXISTS "honorSksRate" (
    "id"            uuid PRIMARY KEY,
    "berlakuMulai"  date NOT NULL UNIQUE,
    "perSks"        double precision NOT NULL CHECK ("perSks" >= 0),
    "aktif"         boolean NOT NULL DEFAULT true,
    "createdAt"     timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"     timestamp(3) NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS "honorSksPeriode" (
    "id"                uuid PRIMARY KEY,
    "userId"            uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "periodeAkademikId" uuid NOT NULL REFERENCES "periodeAkademik" ("id"),
    "sks"               numeric(6,2) NOT NULL CHECK ("sks" >= 0),
    "ratePerSks"        double precision NOT NULL CHECK ("ratePerSks" >= 0),
    "totalHonor"        double precision NOT NULL CHECK ("totalHonor" >= 0),
    "skId"              uuid REFERENCES "suratKeputusan" ("id") ON DELETE SET NULL,
    "createdAt"         timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"         timestamp(3) NOT NULL DEFAULT now(),
    UNIQUE ("userId", "periodeAkademikId")
);
CREATE INDEX IF NOT EXISTS honor_sks_periode_user_idx ON "honorSksPeriode" ("userId");

-- ═══════════ 5. BPJS Kesehatan / Ketenagakerjaan ═══════════

-- Konfigurasi iuran per jenis dan periode. Komponen aktual per pegawai tetap
-- disimpan di `komponenPotongan` (nilai historis per pegawai); tabel ini
-- adalah sumber tarif resmi yang berlaku pada suatu periode.
CREATE TABLE IF NOT EXISTS "bpjsKonfigurasi" (
    "id"                     uuid PRIMARY KEY,
    "jenis"                  "JenisBPJS" NOT NULL,
    "iuranPegawaiPct"        numeric(7,4) NOT NULL DEFAULT 0,
    "iuranPemberiKerjaPct"   numeric(7,4) NOT NULL DEFAULT 0,
    "dasar"                  text NOT NULL DEFAULT 'GAJI_POKOK' CHECK ("dasar" IN ('GAJI_POKOK', 'TOTAL_GAJI')),
    "batasAtas"              double precision,
    "berlakuMulai"           date NOT NULL,
    "aktif"                  boolean NOT NULL DEFAULT true,
    "createdAt"              timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"              timestamp(3) NOT NULL DEFAULT now(),
    UNIQUE ("jenis", "berlakuMulai")
);
CREATE TABLE IF NOT EXISTS "bpjsPeserta" (
    "id"                uuid PRIMARY KEY,
    "userId"            uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "jenis"             "JenisBPJS" NOT NULL,
    "nomorKepesertaan"  text,
    "mulai"             date NOT NULL,
    "selesai"           date,
    "aktif"             boolean NOT NULL DEFAULT true,
    "createdAt"         timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"         timestamp(3) NOT NULL DEFAULT now(),
    UNIQUE ("userId", "jenis")
);
-- Catatan: `nomorKepesertaan` mengandung identitas pribadi; perlakukan sebagai
-- data pribadi — batasi pembacaan, redaksi log, dan pertimbangkan enkripsi
-- level aplikasi seperti PII lain bila disetujui (DEC-053/DEC-054).

-- ═══════════ 6. PPh 21 TER (PMK 168/2023) ═══════════

-- Tabel tarif efektif bulanan TER A/B/C berdasarkan PTKP. Pemetaan status
-- PTKP (TK/0, K/0, dst.) → kategori TER dilakukan di lapisan aplikasi.
CREATE TABLE IF NOT EXISTS "pph21TerTarif" (
    "id"            uuid PRIMARY KEY,
    "kategori"      "PtkpKategoriTER" NOT NULL,
    "lapisan"       integer NOT NULL CHECK ("lapisan" >= 1),
    "batasBawah"    double precision NOT NULL CHECK ("batasBawah" >= 0),
    "batasAtas"     double precision,
    "tarifPct"      double precision NOT NULL CHECK ("tarifPct" >= 0 AND "tarifPct" <= 100),
    "berlakuMulai"  date NOT NULL,
    "createdAt"     timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"     timestamp(3) NOT NULL DEFAULT now(),
    UNIQUE ("kategori", "lapisan", "berlakuMulai"),
    CHECK ("batasAtas" IS NULL OR "batasAtas" > "batasBawah")
);
CREATE INDEX IF NOT EXISTS pph21_ter_tarif_berlaku_idx ON "pph21TerTarif" ("berlakuMulai");

-- ═══════════ 7. THR ═══════════

-- Konfigurasi THR per tahun. Besaran 'SATU_GAJI_POKOK'/'GAJI_KOTOR' memakai
-- slip terakhir yang TERBIT; 'PERSEN' mengalikan parameter `persen` dari
-- gaji pokok; 'PERSEN_TAKE_HOME' (`DEC-086`) mengalikan parameter `persen`
-- dari take-home pay (gaji bersih) pada slip TERBIT/TERBAYAR terakhir.
CREATE TABLE IF NOT EXISTS "thrKonfigurasi" (
    "id"                     uuid PRIMARY KEY,
    "tahun"                  integer NOT NULL UNIQUE CHECK ("tahun" >= 2026),
    "besaranTipe"            text NOT NULL CHECK ("besaranTipe" IN ('SATU_GAJI_POKOK', 'GAJI_KOTOR', 'PERSEN', 'PERSEN_TAKE_HOME')),
    "persen"                 double precision CHECK ("persen" BETWEEN 0 AND 400),
    "syaratMasaKerjaBulan"   integer NOT NULL DEFAULT 3 CHECK ("syaratMasaKerjaBulan" >= 0),
    "berlakuTanggal"         date NOT NULL,
    "createdAt"              timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"              timestamp(3) NOT NULL DEFAULT now(),
    CHECK (("besaranTipe" IN ('PERSEN', 'PERSEN_TAKE_HOME')) = ("persen" IS NOT NULL))
);

-- ═══════════ 8. Kontrak kerja (PKWT/PKWTT) & pesangon ═══════════

-- Masa kerja kontrak/perpanjangan memvalidasi batas PKWT (PP 35/2021) di
-- lapisan aplikasi; `selesai` NULL berarti PKWTT. Alert perpanjangan
-- dihitung dari `selesai` (pekerjaan terjadwal).
CREATE TABLE IF NOT EXISTS "kontrakKerja" (
    "id"            uuid PRIMARY KEY,
    "userId"        uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "jenis"         "JenisKontrak" NOT NULL,
    "nomorSk"       text NOT NULL UNIQUE,
    "mulai"         date NOT NULL,
    "selesai"       date,
    "status"        text NOT NULL DEFAULT 'AKTIF' CHECK ("status" IN ('AKTIF', 'BERAKHIR', 'DIPERPANJANG')),
    "skId"          uuid REFERENCES "suratKeputusan" ("id") ON DELETE SET NULL,
    "createdAt"     timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"     timestamp(3) NOT NULL DEFAULT now(),
    CHECK ("selesai" IS NULL OR "selesai" > "mulai")
);
CREATE INDEX IF NOT EXISTS kontrak_kerja_user_idx ON "kontrakKerja" ("userId");
CREATE INDEX IF NOT EXISTS kontrak_kerja_selesai_idx ON "kontrakKerja" ("selesai") WHERE "selesai" IS NOT NULL;

-- Faktor pesangon/uang penghargaan per tahun masa kerja per kategori status.
-- Angka disalin dari PP 35/2021 (dan peraturan yayasan yang disetujui);
-- bukan tarif tetap dari kode.
CREATE TABLE IF NOT EXISTS "pesangonAturan" (
    "id"                  uuid PRIMARY KEY,
    "kategoriStatusId"    uuid NOT NULL REFERENCES "kategoriStatusPegawai" ("id"),
    "masaKerjaTahun"      integer NOT NULL CHECK ("masaKerjaTahun" >= 0),
    "pesangonBulan"       double precision NOT NULL DEFAULT 0 CHECK ("pesangonBulan" >= 0),
    "penghargaanBulan"    double precision NOT NULL DEFAULT 0 CHECK ("penghargaanBulan" >= 0),
    "gantiKerugianBulan"  double precision NOT NULL DEFAULT 0 CHECK ("gantiKerugianBulan" >= 0),
    "sumberRegulasi"      text NOT NULL DEFAULT 'PP 35/2021',
    "createdAt"           timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"           timestamp(3) NOT NULL DEFAULT now(),
    UNIQUE ("kategoriStatusId", "masaKerjaTahun")
);

-- ═══════════ 9. Pensiun (BUP yayasan + DPLK) ═══════════

-- BUP ditentukan yayasan per kategori status; DPLK menggantikan skema BKN.
-- Tanggal mulai pensiun dihitung saat penetapan (tanggal lahir + BUP) dan
-- disimpan di `pensiunKepesertaan` agar dapat dipantau penjadwal.
CREATE TABLE IF NOT EXISTS "pensiunKonfigurasi" (
    "id"                     uuid PRIMARY KEY,
    "kategoriStatusId"       uuid NOT NULL REFERENCES "kategoriStatusPegawai" ("id"),
    "bupUmur"                integer NOT NULL CHECK ("bupUmur" BETWEEN 45 AND 80),
    "dplkWajib"              boolean NOT NULL DEFAULT true,
    "iuranPegawaiPct"        numeric(7,4) NOT NULL DEFAULT 0,
    "iuranPemberiKerjaPct"   numeric(7,4) NOT NULL DEFAULT 0,
    "penyediaDplk"           text,
    "berlakuMulai"           date NOT NULL,
    "aktif"                  boolean NOT NULL DEFAULT true,
    "createdAt"              timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"              timestamp(3) NOT NULL DEFAULT now(),
    UNIQUE ("kategoriStatusId", "berlakuMulai")
);
CREATE TABLE IF NOT EXISTS "pensiunKepesertaan" (
    "id"                 uuid PRIMARY KEY,
    "userId"             uuid NOT NULL UNIQUE REFERENCES "user" ("id") ON DELETE CASCADE,
    "penyediaDplk"       text NOT NULL,
    "nomorKepesertaan"   text,
    "tanggalMulaiPensiun" date,
    "mulai"              date NOT NULL,
    "aktif"              boolean NOT NULL DEFAULT true,
    "createdAt"          timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"          timestamp(3) NOT NULL DEFAULT now()
);

-- ═══════════ RLS default-deny untuk seluruh tabel usulan ═══════════

ALTER TABLE "kategoriStatusPegawai" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "skalaGaji" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "skalaGajiDetail" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "suratKeputusan" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "honorSksRate" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "honorSksPeriode" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "bpjsKonfigurasi" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "bpjsPeserta" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "pph21TerTarif" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "thrKonfigurasi" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "kontrakKerja" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "pesangonAturan" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "pensiunKonfigurasi" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "pensiunKepesertaan" ENABLE ROW LEVEL SECURITY;

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "pensiunKepesertaan";
DROP TABLE IF EXISTS "pensiunKonfigurasi";
DROP TABLE IF EXISTS "pesangonAturan";
DROP TABLE IF EXISTS "kontrakKerja";
DROP TABLE IF EXISTS "thrKonfigurasi";
DROP TABLE IF EXISTS "pph21TerTarif";
DROP TABLE IF EXISTS "bpjsPeserta";
DROP TABLE IF EXISTS "bpjsKonfigurasi";
DROP TABLE IF EXISTS "honorSksPeriode";
DROP TABLE IF EXISTS "honorSksRate";
DROP TABLE IF EXISTS "suratKeputusan";
DROP TABLE IF EXISTS "skalaGajiDetail";
DROP TABLE IF EXISTS "skalaGaji";
ALTER TABLE "user" DROP COLUMN IF EXISTS "kategoriStatusId";
DROP TABLE IF EXISTS "kategoriStatusPegawai";
DROP TYPE IF EXISTS "TingkatSk";
DROP TYPE IF EXISTS "JenisSk";
DROP TYPE IF EXISTS "JenisKontrak";
DROP TYPE IF EXISTS "PtkpKategoriTER";
DROP TYPE IF EXISTS "JenisBPJS";
DROP TYPE IF EXISTS "KategoriStatusPegawai";
-- +goose StatementEnd
