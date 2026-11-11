-- +goose Up
-- Draft only: select and approve a versioned migration runner before applying.
-- Row Level Security diaktifkan pada seluruh tabel aplikasi dengan default deny.
-- Backend Go terhubung memakai role pemilik tabel. Tanpa FORCE pada level tabel,
-- role pemilik tetap melewati RLS sehingga backend tidak terblokir.
-- Akses lewat Supabase Data API hanya sah bila ada policy eksplisit per tabel
-- yang mencerminkan cakupan organisasi pengguna. Policy tersebut belum disetujui
-- dan sengaja tidak disertakan pada migrasi ini.

-- +goose StatementBegin
DO $$
DECLARE
    target text;
BEGIN
    FOREACH target IN ARRAY ARRAY[
        'account',
        'AttendanceEvent',
        'auditLog',
        'Cuti',
        'delegasi',
        'Dokumen',
        'gaji',
        'gajiPotongan',
        'gajiTunjangan',
        'HariLibur',
        'Homebase',
        'JawabanSurvei',
        'kategoriPegawai',
        'KomponenGaji',
        'Lembur',
        'MesinAbsensi',
        'MutasiJabatan',
        'Notifikasi',
        'paklaring',
        'PartisipasiSurvei',
        'Pemberhentian',
        'pengaturanGaji',
        'Pengumuman',
        'Peringatan',
        'PeriodeSakit',
        'PertanyaanSurvei',
        'Pesangon',
        'pesanKeluar',
        'Presensi',
        'RiwayatGaji',
        'RiwayatJabatan',
        'riwayatLogin',
        'RiwayatPendidikan',
        'Role',
        'SaldoCuti',
        'SasaranKinerja',
        'session',
        'Survei',
        'TemplateSidikJari',
        'TerusanCuti',
        'TugasBelajar',
        'twoFactor',
        'UjiKapabilitas',
        'user',
        'verification'
    ] LOOP
        IF to_regclass(format('%I', target)) IS NOT NULL THEN
            EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', target);
        END IF;
    END LOOP;
END $$;
-- +goose StatementEnd

-- Policy eksplisit per tabel ditambahkan terpisah setelah disetujui pemilik
-- kebijakan. Tanpa policy, role anon dan authenticated mendapat penolakan
-- otomatis untuk seluruh tabel aplikasi.
