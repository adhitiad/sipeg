-- +goose Up
-- Draft only: select and approve a versioned migration runner before applying.
-- Fondasi skema awal SIMPEG dan hirarki organisasi direktori.

-- ═══════════════════════════ Enum types ═══════════════════════════

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisKelamin" AS ENUM ('L', 'P');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "StatusPresensi" AS ENUM ('H', 'I', 'S', 'A', 'C', 'LD');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisCuti" AS ENUM (
        'TAHUNAN', 'PANJANG', 'BERSALIN', 'SAKIT',
        'SAKIT_BERKEPANJANGAN', 'IZIN_PENTING', 'POLITIK'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "SubJenisIzinPenting" AS ENUM (
        'MENIKAH', 'MELAHIRKAN_ISTRI', 'MENIKAHKAN_ANAK',
        'KELUARGA_INTI', 'KELUARGA_LAIN'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "StatusPersetujuan" AS ENUM ('MENUNGGU', 'DISETUJUI', 'DITOLAK');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "TingkatTerusan" AS ENUM ('PIMPINAN', 'REKTOR', 'YAYASAN');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "StatusSlipGaji" AS ENUM ('DRAF', 'TERBIT', 'TERBAYAR');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "HasilUjiKapabilitas" AS ENUM ('LAYAK_KEMBALI', 'BERHENTI_HORMAT');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "ArahKomponenGaji" AS ENUM ('TUNJANGAN', 'POTONGAN');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisJabatan" AS ENUM ('STRUKTURAL', 'FUNGSIONAL');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisOrganisasi" AS ENUM ('UNIT', 'FAKULTAS', 'JURUSAN', 'BIRO', 'BAGIAN');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "CakupanDelegasi" AS ENUM ('CUTI', 'LEMBUR', 'SEMUA');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisMutasi" AS ENUM ('PROMOSI', 'MUTASI', 'ROTASI', 'DEMOSI');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "TingkatPeringatan" AS ENUM ('LISAN', 'TERTULIS_I', 'TERTULIS_II', 'TERTULIS_III');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisSanksi" AS ENUM ('TEGURAN', 'PENUNDAAN_KGB', 'POTONG_TUNJANGAN');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "JenisPemberhentian" AS ENUM ('BUP', 'ATAS_PERMINTAAN_SENDIRI', 'HORMAT', 'TIDAK_HORMAT');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "KanalPesan" AS ENUM ('WA', 'EMAIL', 'INAPP');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE "StatusPesan" AS ENUM ('DRAF', 'ANTREAN', 'TERKIRIM', 'GAGAL');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- Schema app untuk fungsi helper RLS
CREATE SCHEMA IF NOT EXISTS app;

-- ═══════════════════════════ Tabel Inti ═══════════════════════════

CREATE TABLE IF NOT EXISTS "Role" (
    "id"          uuid PRIMARY KEY,
    "name"        text NOT NULL UNIQUE,
    "description" text,
    "createdAt"   timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"   timestamp(3) NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "Homebase" (
    "id"          uuid PRIMARY KEY,
    "name"        text NOT NULL UNIQUE,
    "description" text,
    "latitude"    double precision,
    "longitude"   double precision,
    "radiusM"     integer,
    "createdAt"   timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"   timestamp(3) NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "kategoriPegawai" (
    "id"          uuid PRIMARY KEY,
    "kode"        text NOT NULL UNIQUE,
    "nama"        text NOT NULL,
    "usiaPensiun" integer NOT NULL,
    "aktif"       boolean NOT NULL DEFAULT true,
    "createdAt"   timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"   timestamp(3) NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "user" (
    "id"                  uuid PRIMARY KEY,
    "email"               text NOT NULL UNIQUE,
    "emailVerified"       boolean NOT NULL DEFAULT false,
    "image"               text,
    "nama"                text NOT NULL,
    "nik"                 text UNIQUE,
    "nip"                 text UNIQUE,
    "jenis_kelamin"       "JenisKelamin",
    "tanggal_lahir"       timestamp(3),
    "no_hp"               text,
    "no_wa"               text,
    "alamat"              text,
    "desa"                text,
    "kecamatan"           text,
    "kota"                text,
    "provinsi"            text,
    "jabatan_struktural"  text,
    "jabatan_fungsional"  text,
    "kategoriPegawaiId"   uuid REFERENCES "kategoriPegawai" ("id"),
    "bankNama"            text,
    "bankRekening"        text,
    "bankAtasNama"        text,
    "tanggalMasuk"        timestamp(3),
    "gajiPokok"           double precision,
    "npwp"                text,
    "statusPtkp"          text DEFAULT 'TK/0',
    "no_kk"               text,
    "no_kk_encrypted"     boolean NOT NULL DEFAULT false,
    "roleId"              uuid REFERENCES "Role" ("id"),
    "homebaseId"          uuid REFERENCES "Homebase" ("id"),
    "twoFactorEnabled"    boolean DEFAULT false,
    "createdAt"           timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"           timestamp(3) NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS "user_homebase_idx" ON "user" ("homebaseId");

-- ═══════════════════════════ Better Auth ═══════════════════════════

CREATE TABLE IF NOT EXISTS "session" (
    "id"        uuid PRIMARY KEY,
    "userId"    uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "expiresAt" timestamp(3) NOT NULL,
    "token"     text NOT NULL UNIQUE,
    "ipAddress" text,
    "userAgent" text,
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt" timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "session_user_idx" ON "session" ("userId");

CREATE TABLE IF NOT EXISTS "account" (
    "id"                    uuid PRIMARY KEY,
    "userId"                uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "providerId"            text NOT NULL,
    "accountId"             text NOT NULL,
    "accessToken"           text,
    "refreshToken"          text,
    "idToken"               text,
    "accessTokenExpiresAt"  timestamp(3),
    "refreshTokenExpiresAt" timestamp(3),
    "scope"                 text,
    "password"              text,
    "createdAt"             timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"             timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "account_user_idx" ON "account" ("userId");

CREATE TABLE IF NOT EXISTS "verification" (
    "id"         uuid PRIMARY KEY,
    "identifier" text NOT NULL,
    "value"      text NOT NULL,
    "expiresAt"  timestamp(3) NOT NULL,
    "createdAt"  timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"  timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "verification_identifier_idx" ON "verification" ("identifier");

CREATE TABLE IF NOT EXISTS "twoFactor" (
    "id"                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    "secret"                  text NOT NULL,
    "backupCodes"             text NOT NULL,
    "userId"                  uuid NOT NULL UNIQUE REFERENCES "user" ("id") ON DELETE CASCADE,
    "verified"                boolean DEFAULT true,
    "failedVerificationCount" integer DEFAULT 0,
    "lockedUntil"             timestamp(3)
);
CREATE INDEX IF NOT EXISTS "twoFactor_user_idx" ON "twoFactor" ("userId");

-- ═══════════════════════════ Kepegawaian ═══════════════════════════

CREATE TABLE IF NOT EXISTS "delegasi" (
    "id"             uuid PRIMARY KEY,
    "pemberiUserId"  uuid NOT NULL REFERENCES "user" ("id"),
    "penerimaUserId" uuid NOT NULL REFERENCES "user" ("id"),
    "cakupan"        "CakupanDelegasi" NOT NULL DEFAULT 'CUTI',
    "mulai"          date NOT NULL,
    "selesai"        date NOT NULL,
    "aktif"          boolean NOT NULL DEFAULT true,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "delegasi_penerima_idx" ON "delegasi" ("penerimaUserId", "aktif");

CREATE TABLE IF NOT EXISTS "riwayatLogin" (
    "id"        uuid PRIMARY KEY,
    "userId"    uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "ip"        text,
    "userAgent" text,
    "createdAt" timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "riwayat_login_user_idx" ON "riwayatLogin" ("userId");

CREATE TABLE IF NOT EXISTS "RiwayatJabatan" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id"),
    "jenis"          "JenisJabatan" NOT NULL,
    "jabatan"        text NOT NULL,
    "tanggalMulai"   timestamp(3) NOT NULL,
    "tanggalSelesai" timestamp(3),
    "nomorSk"        text,
    "keterangan"     text,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "riwayat_jabatan_user_idx" ON "RiwayatJabatan" ("userId", "tanggalMulai");

CREATE TABLE IF NOT EXISTS "RiwayatPendidikan" (
    "id"         uuid PRIMARY KEY,
    "userId"     uuid NOT NULL REFERENCES "user" ("id"),
    "jenjang"    text NOT NULL,
    "institusi"  text NOT NULL,
    "jurusan"    text,
    "tahunLulus" integer,
    "keterangan" text,
    "createdAt"  timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"  timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "riwayat_pendidikan_user_idx" ON "RiwayatPendidikan" ("userId");

CREATE TABLE IF NOT EXISTS "RiwayatGaji" (
    "id"          uuid PRIMARY KEY,
    "userId"      uuid NOT NULL REFERENCES "user" ("id"),
    "gajiLama"    double precision,
    "gajiBaru"    double precision NOT NULL,
    "sumber"      text NOT NULL,
    "referensiId" uuid,
    "diubahOleh"  uuid,
    "createdAt"   timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "riwayat_gaji_user_idx" ON "RiwayatGaji" ("userId");

-- ═══════════════════════════ Penggajian ═══════════════════════════

CREATE TABLE IF NOT EXISTS "KomponenGaji" (
    "id"        uuid PRIMARY KEY,
    "nama"      text NOT NULL UNIQUE,
    "arah"      "ArahKomponenGaji" NOT NULL,
    "nominal"   double precision NOT NULL,
    "aktif"     boolean NOT NULL DEFAULT true,
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt" timestamp(3) NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "gaji" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id"),
    "nomorSlip"      text NOT NULL UNIQUE,
    "kodeVerifikasi" text,
    "periode"        date NOT NULL,
    "tanggalGaji"    date NOT NULL,
    "gajiPokok"      double precision NOT NULL,
    "totalGaji"      double precision NOT NULL,
    "status"         "StatusSlipGaji" NOT NULL DEFAULT 'DRAF',
    "pkpSetahun"     double precision,
    "potonganPph21"  double precision NOT NULL DEFAULT 0,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "gaji_user_periode_key" UNIQUE ("userId", "periode")
);
CREATE INDEX IF NOT EXISTS "gaji_periode_idx" ON "gaji" ("periode");
CREATE INDEX IF NOT EXISTS "gaji_kode_verifikasi_idx" ON "gaji" ("kodeVerifikasi");

CREATE TABLE IF NOT EXISTS "gajiTunjangan" (
    "id"        uuid PRIMARY KEY,
    "gajiId"    uuid NOT NULL REFERENCES "gaji" ("id") ON DELETE CASCADE,
    "jenis"     text NOT NULL,
    "nominal"   double precision NOT NULL,
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt" timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "gaji_tunjangan_gaji_idx" ON "gajiTunjangan" ("gajiId");

CREATE TABLE IF NOT EXISTS "gajiPotongan" (
    "id"        uuid PRIMARY KEY,
    "gajiId"    uuid NOT NULL REFERENCES "gaji" ("id") ON DELETE CASCADE,
    "jenis"     text NOT NULL,
    "nominal"   double precision NOT NULL,
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt" timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "gaji_potongan_gaji_idx" ON "gajiPotongan" ("gajiId");

CREATE TABLE IF NOT EXISTS "SasaranKinerja" (
    "id"          uuid PRIMARY KEY,
    "userId"      uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "tahun"       integer NOT NULL,
    "semester"    integer NOT NULL,
    "uraian"      text NOT NULL,
    "target"      double precision NOT NULL,
    "realisasi"   double precision NOT NULL DEFAULT 0,
    "bobot"       double precision NOT NULL DEFAULT 1,
    "dinilaiOleh" uuid,
    "createdAt"   timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"   timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "sasaran_kinerja_user_periode_idx" ON "SasaranKinerja" ("userId", "tahun", "semester");

CREATE TABLE IF NOT EXISTS "pengaturanGaji" (
    "id"            uuid PRIMARY KEY,
    "tanggalGajian" integer NOT NULL DEFAULT 28,
    "createdAt"     timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"     timestamp(3) NOT NULL DEFAULT now()
);

-- ═══════════════════════════ Operasional ═══════════════════════════

CREATE TABLE IF NOT EXISTS "Pengumuman" (
    "id"              uuid PRIMARY KEY,
    "judul"           text NOT NULL,
    "isi"             text NOT NULL,
    "disematkan"      boolean NOT NULL DEFAULT false,
    "diterbitkanOleh" uuid,
    "createdAt"       timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"       timestamp(3) NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "Notifikasi" (
    "id"        uuid PRIMARY KEY,
    "userId"    uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "judul"     text NOT NULL,
    "isi"       text NOT NULL,
    "tautan"    text,
    "dibaca"    boolean NOT NULL DEFAULT false,
    "createdAt" timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "notifikasi_user_idx" ON "Notifikasi" ("userId", "dibaca");

CREATE TABLE IF NOT EXISTS "Lembur" (
    "id"          uuid PRIMARY KEY,
    "userId"      uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "tanggal"     date NOT NULL,
    "menit"       integer NOT NULL,
    "keperluan"   text,
    "status"      "StatusPersetujuan" NOT NULL DEFAULT 'MENUNGGU',
    "disetujuiBy" uuid,
    "catatan"     text,
    "createdAt"   timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"   timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "lembur_user_tanggal_key" UNIQUE ("userId", "tanggal")
);
CREATE INDEX IF NOT EXISTS "lembur_status_idx" ON "Lembur" ("status");

CREATE TABLE IF NOT EXISTS "TugasBelajar" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "jenjang"        text NOT NULL,
    "institusi"      text NOT NULL,
    "jurusan"        text,
    "tanggalMulai"   date NOT NULL,
    "tanggalSelesai" date NOT NULL,
    "ikatanTahun"    integer NOT NULL DEFAULT 2,
    "nomorSk"        text,
    "status"         "StatusPersetujuan" NOT NULL DEFAULT 'MENUNGGU',
    "disetujuiBy"    uuid,
    "catatan"        text,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "tugas_belajar_user_idx" ON "TugasBelajar" ("userId");
CREATE INDEX IF NOT EXISTS "tugas_belajar_status_idx" ON "TugasBelajar" ("status");

CREATE TABLE IF NOT EXISTS "Survei" (
    "id"         uuid PRIMARY KEY,
    "judul"      text NOT NULL,
    "deskripsi"  text,
    "mulai"      timestamp(3) NOT NULL,
    "selesai"    timestamp(3) NOT NULL,
    "aktif"      boolean NOT NULL DEFAULT true,
    "dibuatOleh" uuid,
    "createdAt"  timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"  timestamp(3) NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "PertanyaanSurvei" (
    "id"       uuid PRIMARY KEY,
    "surveiId" uuid NOT NULL REFERENCES "Survei" ("id") ON DELETE CASCADE,
    "teks"     text NOT NULL,
    "urutan"   integer NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS "pertanyaan_survei_idx" ON "PertanyaanSurvei" ("surveiId");

CREATE TABLE IF NOT EXISTS "JawabanSurvei" (
    "id"           uuid PRIMARY KEY,
    "surveiId"     uuid NOT NULL REFERENCES "Survei" ("id") ON DELETE CASCADE,
    "pertanyaanId" uuid NOT NULL REFERENCES "PertanyaanSurvei" ("id") ON DELETE CASCADE,
    "nilai"        integer NOT NULL,
    "createdAt"    timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "jawaban_survei_idx" ON "JawabanSurvei" ("surveiId");

CREATE TABLE IF NOT EXISTS "PartisipasiSurvei" (
    "id"        uuid PRIMARY KEY,
    "surveiId"  uuid NOT NULL REFERENCES "Survei" ("id") ON DELETE CASCADE,
    "userHash"  text NOT NULL,
    "createdAt" timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "partisipasi_survei_key" UNIQUE ("surveiId", "userHash")
);

-- ═══════════════════════════ Presensi ═══════════════════════════

CREATE TABLE IF NOT EXISTS "Presensi" (
    "id"          uuid PRIMARY KEY,
    "userId"      uuid NOT NULL REFERENCES "user" ("id"),
    "fingerprint" text UNIQUE,
    "tanggal"     timestamp(3) NOT NULL,
    "status"      "StatusPresensi" NOT NULL DEFAULT 'H',
    "keterangan"  text,
    "lokasi"      text,
    "sumber"      text,
    "latitude"    double precision,
    "longitude"   double precision,
    "fotoUrl"     text,
    "createdBy"   uuid,
    "updatedBy"   uuid,
    "createdAt"   timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"   timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "presensi_user_tanggal_key" UNIQUE ("userId", "tanggal")
);
CREATE INDEX IF NOT EXISTS "presensi_tanggal_idx" ON "Presensi" ("tanggal");

CREATE TABLE IF NOT EXISTS "MesinAbsensi" (
    "id"              uuid PRIMARY KEY,
    "nama"            text NOT NULL,
    "merek"           text NOT NULL,
    "host"            text,
    "port"            integer,
    "comKey"          integer,
    "aktif"           boolean NOT NULL DEFAULT true,
    "terakhirSinkron" timestamp(3),
    "createdAt"       timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"       timestamp(3) NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "TemplateSidikJari" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "jariKe"         integer NOT NULL,
    "template"       text NOT NULL,
    "mesinSumber"    text,
    "diperbaruiOleh" uuid,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "template_user_jari_key" UNIQUE ("userId", "jariKe")
);
CREATE INDEX IF NOT EXISTS "template_user_idx" ON "TemplateSidikJari" ("userId");

-- ═══════════════════════════ Cuti ═══════════════════════════

CREATE TABLE IF NOT EXISTS "Cuti" (
    "id"                    uuid PRIMARY KEY,
    "userId"                uuid NOT NULL REFERENCES "user" ("id"),
    "jenis"                 "JenisCuti" NOT NULL,
    "tanggalMulai"          timestamp(3) NOT NULL,
    "tanggalSelesai"        timestamp(3) NOT NULL,
    "alamatCuti"            text,
    "alasan"                text,
    "status"                "StatusPersetujuan" NOT NULL DEFAULT 'MENUNGGU',
    "disetujuiBy"           uuid,
    "catatan"               text,
    "subJenis"              "SubJenisIzinPenting",
    "nomorSurat"            text,
    "tanggalSuratDokter"    timestamp(3),
    "fileSuratUrl"          text,
    "persenGaji"            double precision NOT NULL DEFAULT 100,
    "untukHaji"             boolean NOT NULL DEFAULT false,
    "tahapSaatIni"          "TingkatTerusan",
    "tanggalPerkiraanLahir" timestamp(3),
    "tanggalLahirAktual"    timestamp(3),
    "terpilih"              boolean NOT NULL DEFAULT false,
    "masaJabatanMulai"      timestamp(3),
    "masaJabatanSelesai"    timestamp(3),
    "createdBy"             uuid,
    "updatedBy"             uuid,
    "createdAt"             timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"             timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "cuti_user_tanggal_idx" ON "Cuti" ("userId", "tanggalMulai");
CREATE INDEX IF NOT EXISTS "cuti_status_idx" ON "Cuti" ("status");

CREATE TABLE IF NOT EXISTS "TerusanCuti" (
    "id"         uuid PRIMARY KEY,
    "cutiId"     uuid NOT NULL REFERENCES "Cuti" ("id") ON DELETE CASCADE,
    "dariUserId" uuid NOT NULL,
    "kepada"     "TingkatTerusan" NOT NULL,
    "catatan"    text,
    "createdAt"  timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "terusan_cuti_idx" ON "TerusanCuti" ("cutiId");

CREATE TABLE IF NOT EXISTS "SaldoCuti" (
    "id"           uuid PRIMARY KEY,
    "userId"       uuid NOT NULL REFERENCES "user" ("id"),
    "tahun"        integer NOT NULL,
    "kuotaTahunan" integer NOT NULL DEFAULT 12,
    "terpakai"     integer NOT NULL DEFAULT 0,
    "sisaCarry"    integer NOT NULL DEFAULT 0,
    "hangus"       boolean NOT NULL DEFAULT false,
    "createdAt"    timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"    timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "saldo_cuti_user_tahun_key" UNIQUE ("userId", "tahun")
);
CREATE INDEX IF NOT EXISTS "saldo_cuti_user_idx" ON "SaldoCuti" ("userId");

CREATE TABLE IF NOT EXISTS "PeriodeSakit" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id"),
    "tanggalMulai"   timestamp(3) NOT NULL,
    "bulanKe"        integer NOT NULL DEFAULT 1,
    "persenBerjalan" double precision NOT NULL DEFAULT 100,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "periode_sakit_user_idx" ON "PeriodeSakit" ("userId", "tanggalMulai");

-- ═══════════════════════════ Lifecycle ═══════════════════════════

CREATE TABLE IF NOT EXISTS "UjiKapabilitas" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id"),
    "periodeSakitId" uuid REFERENCES "PeriodeSakit" ("id"),
    "bulanKe"        integer NOT NULL,
    "hasil"          "HasilUjiKapabilitas",
    "tanggalUji"     timestamp(3),
    "diputusBy"      uuid,
    "catatan"        text,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "uji_kapabilitas_user_idx" ON "UjiKapabilitas" ("userId");
CREATE INDEX IF NOT EXISTS "uji_kapabilitas_hasil_idx" ON "UjiKapabilitas" ("hasil");

CREATE TABLE IF NOT EXISTS "MutasiJabatan" (
    "id"                  uuid PRIMARY KEY,
    "userId"              uuid NOT NULL REFERENCES "user" ("id"),
    "jenis"               "JenisMutasi" NOT NULL,
    "jabatanLama"         text,
    "homebaseLama"        text,
    "jabatanBaru"         text NOT NULL,
    "jenisJabatanBaru"    "JenisJabatan" NOT NULL,
    "homebaseBaru"        text,
    "homebaseBaruId"      uuid,
    "gajiPokokBaru"       double precision,
    "nomorSk"             text,
    "tanggalEfektif"      timestamp(3) NOT NULL,
    "alasan"              text,
    "status"              "StatusPersetujuan" NOT NULL DEFAULT 'MENUNGGU',
    "disetujuiBy"         uuid,
    "catatan"             text,
    "disetujuiAtasanBy"   uuid,
    "disetujuiAtasanPada" timestamp(3),
    "createdAt"           timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"           timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "mutasi_user_tanggal_idx" ON "MutasiJabatan" ("userId", "tanggalEfektif");
CREATE INDEX IF NOT EXISTS "mutasi_status_idx" ON "MutasiJabatan" ("status");

CREATE TABLE IF NOT EXISTS "Dokumen" (
    "id"            uuid PRIMARY KEY,
    "userId"        uuid NOT NULL REFERENCES "user" ("id"),
    "judul"         text NOT NULL,
    "jenis"         text,
    "nomor"         text,
    "tanggal"       timestamp(3),
    "fileUrl"       text NOT NULL,
    "keterangan"    text,
    "berlakuSampai" timestamp(3),
    "createdBy"     uuid,
    "updatedBy"     uuid,
    "createdAt"     timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"     timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "dokumen_user_idx" ON "Dokumen" ("userId");

CREATE TABLE IF NOT EXISTS "Peringatan" (
    "id"            uuid PRIMARY KEY,
    "userId"        uuid NOT NULL REFERENCES "user" ("id"),
    "tingkat"       "TingkatPeringatan" NOT NULL,
    "alasan"        text NOT NULL,
    "nomorSurat"    text,
    "tanggal"       date NOT NULL,
    "penerbit"      text NOT NULL,
    "sanksi"        "JenisSanksi",
    "sanksiCatatan" text,
    "createdBy"     uuid,
    "createdAt"     timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"     timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "peringatan_user_tanggal_idx" ON "Peringatan" ("userId", "tanggal");

CREATE TABLE IF NOT EXISTS "Pemberhentian" (
    "id"               uuid PRIMARY KEY,
    "userId"           uuid NOT NULL REFERENCES "user" ("id"),
    "jenis"            "JenisPemberhentian" NOT NULL,
    "tanggal"          date NOT NULL,
    "nomorSk"          text NOT NULL,
    "hakTerakhir"      text,
    "ujiKapabilitasId" uuid REFERENCES "UjiKapabilitas" ("id"),
    "keterangan"       text,
    "createdBy"        uuid,
    "createdAt"        timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"        timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "pemberhentian_user_tanggal_idx" ON "Pemberhentian" ("userId", "tanggal");
CREATE INDEX IF NOT EXISTS "pemberhentian_jenis_idx" ON "Pemberhentian" ("jenis");
CREATE INDEX IF NOT EXISTS "pemberhentian_uji_idx" ON "Pemberhentian" ("ujiKapabilitasId");

CREATE TABLE IF NOT EXISTS "Pesangon" (
    "id"             uuid PRIMARY KEY,
    "userId"         uuid NOT NULL REFERENCES "user" ("id"),
    "tanggalPensiun" date NOT NULL,
    "masaKerjaBulan" integer NOT NULL,
    "gajiPokok"      double precision NOT NULL,
    "kelipatan"      integer NOT NULL,
    "nominal"        double precision NOT NULL,
    "nomorSk"        text,
    "diterbitkanBy"  uuid,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "pesangon_user_idx" ON "Pesangon" ("userId");

CREATE TABLE IF NOT EXISTS "paklaring" (
    "id"                     uuid PRIMARY KEY,
    "userId"                 uuid NOT NULL REFERENCES "user" ("id"),
    "nomorSurat"             text NOT NULL,
    "keperluan"              text,
    "tanggalTerbit"          date NOT NULL,
    "namaSnapshot"           text NOT NULL,
    "nipSnapshot"            text,
    "jabatanSnapshot"        text NOT NULL,
    "unitSnapshot"           text,
    "tanggalMasukSnapshot"   date,
    "tanggalSelesaiSnapshot" date,
    "diterbitkanBy"          uuid,
    "kodeVerifikasi"         text NOT NULL UNIQUE,
    "createdAt"              timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"              timestamp(3) NOT NULL DEFAULT now(),
    CONSTRAINT "paklaring_nomor_key" UNIQUE ("nomorSurat")
);
CREATE INDEX IF NOT EXISTS "paklaring_user_idx" ON "paklaring" ("userId");

CREATE TABLE IF NOT EXISTS "pesanKeluar" (
    "id"             uuid PRIMARY KEY,
    "kanal"          "KanalPesan" NOT NULL,
    "status"         "StatusPesan" NOT NULL DEFAULT 'DRAF',
    "penerimaUserId" uuid,
    "penerima"       text NOT NULL,
    "judul"          text,
    "isi"            text NOT NULL,
    "referensi"      text,
    "dibuatOleh"     uuid,
    "terkirimPada"   timestamp(3),
    "galat"          text,
    "percobaan"      integer NOT NULL DEFAULT 0,
    "createdAt"      timestamp(3) NOT NULL DEFAULT now(),
    "updatedAt"      timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "pesan_keluar_status_idx" ON "pesanKeluar" ("status");
CREATE INDEX IF NOT EXISTS "pesan_keluar_kanal_status_idx" ON "pesanKeluar" ("kanal", "status");

CREATE TABLE IF NOT EXISTS "auditLog" (
    "id"         uuid PRIMARY KEY,
    "aktorId"    uuid,
    "aksi"       text NOT NULL,
    "sasaran"    text,
    "keterangan" text,
    "ip"         text,
    "createdAt"  timestamp(3) NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS "audit_log_aktor_idx" ON "auditLog" ("aktorId");
CREATE INDEX IF NOT EXISTS "audit_log_aksi_idx" ON "auditLog" ("aksi");

-- ═══════════════════════════ Hierarki Direktori ═══════════════════════════

ALTER TABLE "Homebase"
    ADD COLUMN IF NOT EXISTS "jenis" "JenisOrganisasi" NOT NULL DEFAULT 'UNIT',
    ADD COLUMN IF NOT EXISTS "parentId" uuid,
    ADD COLUMN IF NOT EXISTS "aktif" boolean NOT NULL DEFAULT true;

ALTER TABLE "user"
    ADD COLUMN IF NOT EXISTS "aktif" boolean NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS homebase_parent_idx ON "Homebase" ("parentId");

ALTER TABLE "Homebase"
    ADD CONSTRAINT "Homebase_parentId_fkey"
    FOREIGN KEY ("parentId") REFERENCES "Homebase" ("id");

ALTER TABLE "Homebase"
    ADD CONSTRAINT "Homebase_parent_not_self_check"
    CHECK ("parentId" IS NULL OR "parentId" <> "id");

-- +goose Down
DROP TABLE IF EXISTS "auditLog" CASCADE;
DROP TABLE IF EXISTS "pesanKeluar" CASCADE;
DROP TABLE IF EXISTS "paklaring" CASCADE;
DROP TABLE IF EXISTS "Pesangon" CASCADE;
DROP TABLE IF EXISTS "Pemberhentian" CASCADE;
DROP TABLE IF EXISTS "Peringatan" CASCADE;
DROP TABLE IF EXISTS "Dokumen" CASCADE;
DROP TABLE IF EXISTS "MutasiJabatan" CASCADE;
DROP TABLE IF EXISTS "UjiKapabilitas" CASCADE;
DROP TABLE IF EXISTS "PeriodeSakit" CASCADE;
DROP TABLE IF EXISTS "SaldoCuti" CASCADE;
DROP TABLE IF EXISTS "TerusanCuti" CASCADE;
DROP TABLE IF EXISTS "Cuti" CASCADE;
DROP TABLE IF EXISTS "TemplateSidikJari" CASCADE;
DROP TABLE IF EXISTS "MesinAbsensi" CASCADE;
DROP TABLE IF EXISTS "Presensi" CASCADE;
DROP TABLE IF EXISTS "PartisipasiSurvei" CASCADE;
DROP TABLE IF EXISTS "JawabanSurvei" CASCADE;
DROP TABLE IF EXISTS "PertanyaanSurvei" CASCADE;
DROP TABLE IF EXISTS "Survei" CASCADE;
DROP TABLE IF EXISTS "TugasBelajar" CASCADE;
DROP TABLE IF EXISTS "Lembur" CASCADE;
DROP TABLE IF EXISTS "Notifikasi" CASCADE;
DROP TABLE IF EXISTS "Pengumuman" CASCADE;
DROP TABLE IF EXISTS "pengaturanGaji" CASCADE;
DROP TABLE IF EXISTS "SasaranKinerja" CASCADE;
DROP TABLE IF EXISTS "gajiPotongan" CASCADE;
DROP TABLE IF EXISTS "gajiTunjangan" CASCADE;
DROP TABLE IF EXISTS "gaji" CASCADE;
DROP TABLE IF EXISTS "KomponenGaji" CASCADE;
DROP TABLE IF EXISTS "RiwayatGaji" CASCADE;
DROP TABLE IF EXISTS "RiwayatPendidikan" CASCADE;
DROP TABLE IF EXISTS "RiwayatJabatan" CASCADE;
DROP TABLE IF EXISTS "riwayatLogin" CASCADE;
DROP TABLE IF EXISTS "delegasi" CASCADE;
DROP TABLE IF EXISTS "twoFactor" CASCADE;
DROP TABLE IF EXISTS "verification" CASCADE;
DROP TABLE IF EXISTS "account" CASCADE;
DROP TABLE IF EXISTS "session" CASCADE;
DROP TABLE IF EXISTS "user" CASCADE;
DROP TABLE IF EXISTS "kategoriPegawai" CASCADE;
DROP TABLE IF EXISTS "Homebase" CASCADE;
DROP TABLE IF EXISTS "Role" CASCADE;
DROP TYPE IF EXISTS "StatusPesan";
DROP TYPE IF EXISTS "KanalPesan";
DROP TYPE IF EXISTS "JenisPemberhentian";
DROP TYPE IF EXISTS "JenisSanksi";
DROP TYPE IF EXISTS "TingkatPeringatan";
DROP TYPE IF EXISTS "JenisMutasi";
DROP TYPE IF EXISTS "CakupanDelegasi";
DROP TYPE IF EXISTS "JenisOrganisasi";
DROP TYPE IF EXISTS "JenisJabatan";
DROP TYPE IF EXISTS "ArahKomponenGaji";
DROP TYPE IF EXISTS "HasilUjiKapabilitas";
DROP TYPE IF EXISTS "StatusSlipGaji";
DROP TYPE IF EXISTS "TingkatTerusan";
DROP TYPE IF EXISTS "StatusPersetujuan";
DROP TYPE IF EXISTS "SubJenisIzinPenting";
DROP TYPE IF EXISTS "JenisCuti";
DROP TYPE IF EXISTS "StatusPresensi";
DROP TYPE IF EXISTS "JenisKelamin";
DROP SCHEMA IF EXISTS app;
