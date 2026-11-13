package repository

import (
	"fmt"

	"simpeg/backend/internal/crypto"
	"simpeg/backend/internal/models"
)

// setIdentityHashes mengisi kolom blind index (nikHash, nipHash, npwpHash)
// dari nilai plaintext pada model sebelum baris disimpan. Nilai kosong
// menghasilkan kolom NULL sehingga indeks unik parsial tidak terganggu.
func setIdentityHashes(user *models.User) error {
	var err error
	if user.NikHash, err = crypto.BlindIndexPtr(user.Nik); err != nil {
		return fmt.Errorf("blind index NIK: %w", err)
	}
	if user.NipHash, err = crypto.BlindIndexPtr(user.Nip); err != nil {
		return fmt.Errorf("blind index NIP: %w", err)
	}
	if user.NpwpHash, err = crypto.BlindIndexPtr(user.Npwp); err != nil {
		return fmt.Errorf("blind index NPWP: %w", err)
	}
	return nil
}

// setKeluargaNikHash mengisi blind index (nikHash) dari NIK plaintext
// keluarga sebelum baris disimpan. Kolom nik keluarga selalu terisi, sehingga
// hash juga selalu terisi berpasangan dengannya.
func setKeluargaNikHash(item *models.KeluargaPegawai) error {
	hash, err := crypto.BlindIndex(string(item.Nik))
	if err != nil {
		return fmt.Errorf("blind index NIK keluarga: %w", err)
	}
	item.NikHash = &hash
	return nil
}
