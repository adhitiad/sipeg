-- Menambahkan status BATAL pada siklus hidup slip gaji (`DEC-083`):
-- approve pemberhentian membatalkan slip DRAF untuk periode setelah bulan
-- pemberhentian; slip BATAL tidak dapat digenerate ulang maupun dicetak.
-- Di luar ruang lingkup perilaku TERBIT/TERBAYAR.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    ALTER TYPE "StatusSlipGaji" ADD VALUE IF NOT EXISTS 'BATAL';
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose Down
-- PostgreSQL tidak mendukung penghapusan nilai enum; migrasi turun tidak
-- menghapus nilai BATAL.
